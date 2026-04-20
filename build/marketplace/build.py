#!/usr/bin/env python3

import json
import re
import shutil
import tempfile
import urllib.request
import zipfile
import yaml
from pathlib import Path


# 让 YAML dump 时使用 | 块字面量样式输出多行字符串（如 compose 全文）
class LiteralStr(str):
    """标记一个字符串在 YAML 输出时使用 | 字面量块样式"""
    pass


def _literal_str_representer(dumper, data):
    return dumper.represent_scalar("tag:yaml.org,2002:str", data, style="|")


yaml.add_representer(LiteralStr, _literal_str_representer, Dumper=yaml.SafeDumper)


BASE_DIR = Path(__file__).parent
OUTPUT_INDEX = BASE_DIR / "index.json"
OUTPUT_DOWNLOAD = BASE_DIR / "storage"

SOURCE_URL = "https://github.com/1Panel-dev/appstore/archive/refs/heads/dev.zip"

# 构建阶段将 compose 文件中原始网络名替换为此值
# 若修改此值，请同步修改 index.html 中 NETWORK_NAME 默认值
ORIGINAL_NETWORK_NAME = "1panel-network"
NETWORK_NAME = "app-network"

# 系统注入的环境变量，不需要纳入 formFields（与 index.html env 对齐）
SYSTEM_ENV_KEYS = {"CONTAINER_NAME", "APP_NAME", "NETWORK_NAME"}

# 统一小写比对
COMPOSE_FILE_NAMES = ("docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml")
DATA_FILE_NAMES = ("data.yml", "data.yaml")

# compose 插值变量（仅处理 ${VAR} / ${VAR:-default} / ${VAR?} 等大括号形式，
# 1Panel 模板不使用裸 $VAR 形式；扫描前先移除 $$ 转义）
COMPOSE_VAR_RE = re.compile(
    r"\$\{([A-Za-z_][A-Za-z0-9_]*)(?::?([-?+=])([^}]*))?\}"
)


# ---------- 通用 helper ----------

def find_first(directory: Path, names) -> Path | None:
    """在目录下按候选名顺序查找第一个存在的文件（不递归）"""
    for name in names:
        p = directory / name
        if p.is_file():
            return p
    return None


def load_yaml(path: Path) -> dict:
    """加载 YAML 文件，失败返回空字典"""
    try:
        with open(path, "r", encoding="utf-8") as f:
            return yaml.safe_load(f) or {}
    except Exception as e:
        print(f"  [警告] 读取 {path} 失败: {e}")
        return {}


# ---------- 下载 / 解压 ----------

def download_source(url: str, dest: Path):
    """下载源码 zip 到指定路径"""
    print(f"[下载] {url}")
    with urllib.request.urlopen(url) as resp, open(dest, "wb") as f:
        total = int(resp.headers.get("Content-Length", 0))
        downloaded = 0
        chunk_size = 1024 * 64
        while True:
            chunk = resp.read(chunk_size)
            if not chunk:
                break
            f.write(chunk)
            downloaded += len(chunk)
            if total:
                pct = downloaded * 100 // total
                print(f"\r  进度: {pct}% ({downloaded // 1024} KB / {total // 1024} KB)", end="", flush=True)
    print()


def extract_source(zip_path: Path, extract_dir: Path) -> Path:
    """解压 zip，返回解压后的根目录（zip 内顶层目录）"""
    print(f"[解压] {zip_path.name} -> {extract_dir}")
    with zipfile.ZipFile(zip_path, "r") as zf:
        zf.extractall(extract_dir)
        top_dirs = {Path(name).parts[0] for name in zf.namelist() if name.strip("/")}
        if len(top_dirs) == 1:
            return extract_dir / top_dirs.pop()
        return extract_dir


# ---------- 元数据清洗 ----------

def filter_chinese_content(data):
    """递归处理：
    - locales / label / description 多语言字典仅保留 zh / en
    - 移除冗余 *Zh / *En 字段
    - 字符串 description 自动升级为多语言 dict
    """
    if not isinstance(data, dict):
        return data

    result = {}
    for key, value in data.items():
        if key in ("locales", "label", "description") and isinstance(value, dict):
            picked = {k: value[k] for k in ("zh", "en") if k in value}
            result[key] = picked if picked else value
        elif key == "description" and isinstance(value, str):
            result[key] = {"zh": value, "en": value}
        elif key in ("labelZh", "labelEn", "shortDescZh", "shortDescEn"):
            continue
        elif isinstance(value, dict):
            result[key] = filter_chinese_content(value)
        elif isinstance(value, list):
            result[key] = [filter_chinese_content(v) for v in value]
        else:
            result[key] = value

    return result


def merge_additional_properties(data):
    """递归把任意层级的 additionalProperties 展平到父级"""
    if not isinstance(data, dict):
        if isinstance(data, list):
            return [merge_additional_properties(v) for v in data]
        return data

    result = {}
    for key, value in data.items():
        if key == "additionalProperties" and isinstance(value, dict):
            # 展平到父级（当前 result）；同名则被 additionalProperties 覆盖
            for k, v in value.items():
                result[k] = merge_additional_properties(v)
        else:
            result[key] = merge_additional_properties(value)
    return result


# ---------- compose 变量提取与 formFields 合并 ----------

def extract_compose_variables(compose_path: Path) -> dict:
    """扫描单个 compose 文件，返回 {VAR: default_or_None}

    - 仅识别 ${VAR} / ${VAR:-default} 等大括号形式
    - 先去除 $$ 转义，避免干扰
    - 同名变量：首次不带默认值、后续带默认值时补全
    """
    variables: dict = {}
    try:
        text = compose_path.read_text(encoding="utf-8")
    except (UnicodeDecodeError, OSError):
        return variables

    stripped = text.replace("$$", "")
    for match in COMPOSE_VAR_RE.finditer(stripped):
        name, op, val = match.group(1), match.group(2), match.group(3)
        default = val if op in ("-", "=") and val is not None else None
        if name not in variables:
            variables[name] = default
        elif variables[name] is None and default is not None:
            variables[name] = default

    return variables


def merge_form_fields(existing_fields: list, compose_vars: dict) -> list:
    """将 compose 中发现的变量合并到 data.yml 的 formFields

    - data.yml 中已定义的字段保留原样（优先级高）
    - compose 中存在但 data.yml 未定义的字段自动补齐，标记 auto: true
    - 系统注入变量（CONTAINER_NAME、APP_NAME 等）不参与合并
    """
    existing_keys = {
        f["envKey"] for f in existing_fields
        if isinstance(f, dict) and f.get("envKey")
    }
    merged = list(existing_fields)

    for var_name, default in compose_vars.items():
        if var_name in SYSTEM_ENV_KEYS or var_name in existing_keys:
            continue
        merged.append({
            "type": "text",
            "label": {"zh": var_name, "en": var_name},
            "envKey": var_name,
            "default": default if default is not None else "",
            "required": default is None,
            "auto": True,
        })

    return merged


# ---------- 版本级输出（meta.yml + 可选 init.zip） ----------

def build_version_meta(
    app_name: str,
    version: str,
    version_dir: Path,
    compose_path: Path | None,
    form_fields: list,
):
    """生成 meta.yml + 可选 init.zip

    - compose 原文做网络名字面量替换后以 YAML | 块样式写入 meta.yml
    - 除 data.yml / compose 以外的文件打入 init.zip（无则不生成）
    """
    out_dir = OUTPUT_DOWNLOAD / app_name / version
    out_dir.mkdir(parents=True, exist_ok=True)

    # 1) 读取并处理 compose 文本
    # - 网络名字面量替换
    # - 去除每行行尾空格（部分源文件存在尾空格，会让 PyYAML 判断无法往返而 fallback
    #   到双引号样式，破坏可读性；尾空格对 compose 语义无影响）
    # - 统一以换行符结尾，便于 | 块样式输出
    compose_text = ""
    if compose_path is not None:
        try:
            raw = compose_path.read_text(encoding="utf-8")
            if ORIGINAL_NETWORK_NAME in raw:
                raw = raw.replace(ORIGINAL_NETWORK_NAME, NETWORK_NAME)
                print(f"  [网络] {app_name}/{version}/{compose_path.name}: {ORIGINAL_NETWORK_NAME} -> {NETWORK_NAME}")
            compose_text = "\n".join(line.rstrip() for line in raw.splitlines())
            if compose_text and not compose_text.endswith("\n"):
                compose_text += "\n"
        except UnicodeDecodeError:
            print(f"  [警告] compose 文件读取失败（非 UTF-8）：{compose_path}")

    # 2) 收集除 data / compose 外的其他文件，非空才打包
    skip = set(DATA_FILE_NAMES) | set(COMPOSE_FILE_NAMES)
    extras = [
        p for p in sorted(version_dir.rglob("*"))
        if p.is_file() and p.name.lower() not in skip
    ]
    has_init = bool(extras)
    if has_init:
        with zipfile.ZipFile(out_dir / "init.zip", "w", zipfile.ZIP_DEFLATED) as zf:
            for file_path in extras:
                zf.write(file_path, file_path.relative_to(version_dir))

    # 3) 写 meta.yml
    meta = {
        "compose": LiteralStr(compose_text) if compose_text else "",
        "formFields": form_fields,
    }
    if has_init:
        meta["init"] = "init.zip"

    meta_path = out_dir / "meta.yml"
    with open(meta_path, "w", encoding="utf-8") as f:
        yaml.safe_dump(
            meta, f,
            allow_unicode=True,
            sort_keys=False,
            default_flow_style=False,
            width=4096,
        )

    print(f"  [meta] {meta_path.relative_to(BASE_DIR)}" + (" (+init.zip)" if has_init else ""))


# ---------- 主构建流程 ----------

def _load_data_yaml(directory: Path) -> dict:
    """查找并加载目录下的 data.yml/data.yaml：做中英文过滤并就地展平 additionalProperties

    展平放在加载阶段，好处是后续对 formFields 等字段的 pop / 读取不会漏掉被藏在
    additionalProperties 里的情况。
    """
    path = find_first(directory, DATA_FILE_NAMES)
    if not path:
        return {}
    return merge_additional_properties(filter_chinese_content(load_yaml(path)))


def _copy_app_static_files(app_dir: Path, app_out_dir: Path):
    """复制应用级静态文件：logo.png、README*.md"""
    app_out_dir.mkdir(parents=True, exist_ok=True)
    for src in app_dir.iterdir():
        if not src.is_file():
            continue
        is_logo = src.name == "logo.png"
        is_readme = src.name.startswith("README") and src.suffix.lower() == ".md"
        if is_logo or is_readme:
            shutil.copy2(src, app_out_dir / src.name)
            print(f"  [复制] {app_dir.name}/{src.name}")


def build_index(source_dir: Path) -> dict:
    """构建完整的 index 数据结构"""
    apps_dir = source_dir / "apps"

    # 根层级 data.yaml / data.yml
    index = _load_data_yaml(source_dir)
    if index:
        print("[根] 读取 data.yml")
    index["apps"] = {}

    if not apps_dir.exists():
        print(f"[警告] apps 目录不存在: {apps_dir}")
        return index

    for app_dir in sorted(apps_dir.iterdir()):
        if not app_dir.is_dir():
            continue

        app_name = app_dir.name
        app_entry = _load_data_yaml(app_dir)
        if app_entry:
            print(f"[应用] {app_name}")
        app_entry["versions"] = {}

        _copy_app_static_files(app_dir, OUTPUT_DOWNLOAD / app_name)

        for version_dir in sorted(app_dir.iterdir()):
            if not version_dir.is_dir():
                continue

            version = version_dir.name
            version_entry = _load_data_yaml(version_dir)
            if version_entry:
                print(f"  [版本] {app_name}/{version}")

            # 扫描 compose 文件（一次性查找，后续 meta/变量提取共用）
            compose_path = find_first(version_dir, COMPOSE_FILE_NAMES)
            compose_vars = extract_compose_variables(compose_path) if compose_path else {}

            existing_fields = version_entry.get("formFields") or []
            merged_fields = merge_form_fields(existing_fields, compose_vars) if compose_vars else existing_fields
            auto_keys = [
                f["envKey"] for f in merged_fields
                if isinstance(f, dict) and f.get("auto") and f.get("envKey")
            ]
            if auto_keys:
                print(f"  [变量] 自动补充: {', '.join(auto_keys)}")

            # 输出 meta.yml + 可选 init.zip（formFields 不再回写 index.json）
            build_version_meta(app_name, version, version_dir, compose_path, merged_fields)

            # index.json 只保留轻量版本节点（不含 formFields / compose 内容）
            version_entry.pop("formFields", None)
            app_entry["versions"][version] = version_entry

        index["apps"][app_name] = app_entry

    return index


def main():
    print("=" * 50)
    print("开始构建应用市场...")
    print("=" * 50)

    if OUTPUT_DOWNLOAD.exists():
        shutil.rmtree(OUTPUT_DOWNLOAD)
    OUTPUT_DOWNLOAD.mkdir(parents=True)

    with tempfile.TemporaryDirectory() as tmp:
        tmp_path = Path(tmp)
        zip_path = tmp_path / "applist.zip"

        download_source(SOURCE_URL, zip_path)
        source_dir = extract_source(zip_path, tmp_path / "src")
        print(f"[源码] {source_dir}")

        index = build_index(source_dir)

    with open(OUTPUT_INDEX, "w", encoding="utf-8") as f:
        json.dump(index, f, ensure_ascii=False, indent=2)

    print("=" * 50)
    print("完成！")
    print(f"  index.json -> {OUTPUT_INDEX.relative_to(BASE_DIR)}")
    print(f"  下载包目录  -> {OUTPUT_DOWNLOAD.relative_to(BASE_DIR)}/")
    print("=" * 50)


if __name__ == "__main__":
    main()
