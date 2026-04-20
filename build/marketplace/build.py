#!/usr/bin/env python3

import json
import re
import shutil
import tempfile
import urllib.request
import zipfile
import yaml
from pathlib import Path


BASE_DIR = Path(__file__).parent
OUTPUT_INDEX = BASE_DIR / "index.json"
OUTPUT_DOWNLOAD = BASE_DIR / "storage"

SOURCE_URL = "https://github.com/1Panel-dev/appstore/archive/refs/heads/dev.zip"

# 构建阶段将 compose 文件中原始网络名替换为此值
# 若修改此值，请同步修改 index.html 中 buildInstallScript 的 NETWORK_NAME 默认值
ORIGINAL_NETWORK_NAME = "1panel-network"
NETWORK_NAME = "app-network"

# 系统注入的环境变量，不需要纳入 formFields（与 index.html env.system 对齐）
SYSTEM_ENV_KEYS = {"CONTAINER_NAME", "APP_NAME"}

# compose 插值变量正则：
#   ${VAR}                 - 简单插值
#   ${VAR:-default}        - 默认值（未设置或为空时使用）
#   ${VAR-default}         - 默认值（未设置时使用）
#   ${VAR:?err} / ${VAR?}  - 必填
#   ${VAR:+alt} / ${VAR+}  - 替换值（不作为默认值）
# 使用前先把 $$ 替换掉以避免转义干扰
COMPOSE_VAR_BRACE_RE = re.compile(
    r"\$\{([A-Za-z_][A-Za-z0-9_]*)(?::?([-?+=])([^}]*))?\}"
)
COMPOSE_VAR_PLAIN_RE = re.compile(r"\$([A-Za-z_][A-Za-z0-9_]*)")


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
        # zip 内通常有一个顶层目录，如 master/ 或 dev/ 等
        top_dirs = {Path(name).parts[0] for name in zf.namelist() if name.strip("/")}
        if len(top_dirs) == 1:
            return extract_dir / top_dirs.pop()
        return extract_dir


def load_yaml(path: Path) -> dict:
    """加载 YAML 文件，失败返回空字典"""
    try:
        with open(path, "r", encoding="utf-8") as f:
            return yaml.safe_load(f) or {}
    except Exception as e:
        print(f"  [警告] 读取 {path} 失败: {e}")
        return {}


def filter_chinese_content(data: dict) -> dict:
    """多语言处理，默认保留中文和英文内容，移除冗余字段"""
    if not isinstance(data, dict):
        return data
    
    result = {}
    for key, value in data.items():
        if key in ['locales', 'label', 'description'] and isinstance(value, dict):
            # 多语言字段
            filtered_value = {}
            # 处理中文版本
            if 'zh' in value:
                filtered_value['zh'] = value['zh']
            # 处理英文版本
            if 'en' in value:
                filtered_value['en'] = value['en']
            result[key] = filtered_value if filtered_value else value
        elif key == 'description' and isinstance(value, str):
            # 将单语言 description 转换为多语言格式
            result[key] = {
                'zh': value,
                'en': value
            }
        elif key in ['labelZh', 'labelEn', 'shortDescZh', 'shortDescEn']:
            # 跳过冗余字段
            continue
        elif isinstance(value, dict):
            result[key] = filter_chinese_content(value)
        elif isinstance(value, list):
            result[key] = [filter_chinese_content(item) if isinstance(item, dict) else item for item in value]
        else:
            result[key] = value
    
    return result


def merge_additional_properties(data: dict) -> dict:
    """合并 additionalProperties 到上层"""
    if not isinstance(data, dict):
        return data
    
    result = dict(data)
    
    # 如果存在 additionalProperties，将其内容合并到当前层级
    if 'additionalProperties' in result and isinstance(result['additionalProperties'], dict):
        additional_props = result.pop('additionalProperties')
        # 合并 additionalProperties 的内容，覆盖已有字段
        for key, value in additional_props.items():
            result[key] = value
        
        # 递归处理嵌套的 additionalProperties
        for key, value in result.items():
            if isinstance(value, dict):
                result[key] = merge_additional_properties(value)
            elif isinstance(value, list):
                result[key] = [merge_additional_properties(item) if isinstance(item, dict) else item for item in value]
    else:
        # 递归处理所有嵌套字典
        for key, value in result.items():
            if isinstance(value, dict):
                result[key] = merge_additional_properties(value)
            elif isinstance(value, list):
                result[key] = [merge_additional_properties(item) if isinstance(item, dict) else item for item in value]
    
    return result


def extract_compose_variables(version_dir: Path) -> dict:
    """扫描版本目录下的 docker-compose 文件，提取所有插值变量及其默认值

    返回 {VAR: default_str_or_None}，同名变量以首次出现为准，
    但若后续出现了默认值且首次未带默认值，则补全默认值。
    """
    variables: dict = {}
    compose_names = ("docker-compose.yml", "docker-compose.yaml")

    for file_path in sorted(version_dir.rglob("*")):
        if not file_path.is_file():
            continue
        if file_path.name.lower() not in compose_names:
            continue
        try:
            text = file_path.read_text(encoding="utf-8")
        except UnicodeDecodeError:
            continue

        # 先移除转义的 $$，避免 $$VAR 被误识别
        stripped = text.replace("$$", "")

        # ${VAR:-default} / ${VAR} / ${VAR:?err}
        for match in COMPOSE_VAR_BRACE_RE.finditer(stripped):
            name = match.group(1)
            op = match.group(2)
            val = match.group(3)
            default = val if op in ("-", "=") and val is not None else None
            if name not in variables:
                variables[name] = default
            elif variables[name] is None and default is not None:
                variables[name] = default

        # 去除已匹配的 ${...} 后再扫描 $VAR 形式，避免重复
        plain_text = COMPOSE_VAR_BRACE_RE.sub("", stripped)
        for match in COMPOSE_VAR_PLAIN_RE.finditer(plain_text):
            name = match.group(1)
            if name not in variables:
                variables[name] = None

    return variables


def merge_form_fields(existing_fields: list, compose_vars: dict) -> list:
    """将 compose 中发现的变量合并到 data.yml 的 formFields

    - 以 envKey 为键；data.yml 中已定义的字段保留原样（优先级高）
    - compose 中存在但 data.yml 未定义的字段自动补齐，标记 auto: true
    - 系统注入变量（CONTAINER_NAME、APP_NAME 等）不参与合并
    """
    if not isinstance(existing_fields, list):
        existing_fields = []

    # 现有字段 envKey 集合
    existing_keys = set()
    merged = []
    for field in existing_fields:
        if isinstance(field, dict) and field.get("envKey"):
            existing_keys.add(field["envKey"])
        merged.append(field)

    # 按 compose 中出现顺序追加新字段
    for var_name, default in compose_vars.items():
        if var_name in SYSTEM_ENV_KEYS:
            continue
        if var_name in existing_keys:
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


def build_version_zip(app_name: str, version: str, version_dir: Path):
    """将版本目录中的文件打包为 zip，仅排除 data.yml/data.yaml"""
    zip_dir = OUTPUT_DOWNLOAD / app_name
    zip_dir.mkdir(parents=True, exist_ok=True)
    zip_path = zip_dir / f"{version}.zip"

    compose_names = ("docker-compose.yml", "docker-compose.yaml")

    with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for file_path in sorted(version_dir.rglob("*")):
            if not file_path.is_file():
                continue
            if file_path.name.lower() in ("data.yml", "data.yaml"):
                continue
            arcname = file_path.relative_to(version_dir)
            # compose 文件做网络名字面量替换
            if file_path.name.lower() in compose_names:
                try:
                    text = file_path.read_text(encoding="utf-8")
                except UnicodeDecodeError:
                    zf.write(file_path, arcname)
                    continue
                if ORIGINAL_NETWORK_NAME in text:
                    text = text.replace(ORIGINAL_NETWORK_NAME, NETWORK_NAME)
                    print(f"  [网络] {app_name}/{version}/{arcname}: {ORIGINAL_NETWORK_NAME} -> {NETWORK_NAME}")
                zf.writestr(str(arcname), text)
            else:
                zf.write(file_path, arcname)

    print(f"  [zip] {zip_path.relative_to(BASE_DIR)}")


def build_index(source_dir: Path) -> dict:
    """构建完整的 index 数据结构"""
    apps_dir = source_dir / "apps"

    # 根层级 data.yaml / data.yml
    root_data = {}
    for name in ("data.yaml", "data.yml"):
        candidate = source_dir / name
        if candidate.exists():
            root_data = filter_chinese_content(load_yaml(candidate))
            print(f"[根] 读取 {name}")
            break

    index = dict(root_data)
    index["apps"] = {}

    if not apps_dir.exists():
        print(f"[警告] apps 目录不存在: {apps_dir}")
        return index

    # 遍历每个软件目录
    for app_dir in sorted(apps_dir.iterdir()):
        if not app_dir.is_dir():
            continue

        app_name = app_dir.name
        app_data_file = app_dir / "data.yml"
        if not app_data_file.exists():
            app_data_file = app_dir / "data.yaml"

        app_data = {}
        if app_data_file.exists():
            app_data = filter_chinese_content(load_yaml(app_data_file))
            print(f"[应用] {app_name}")

        app_entry = dict(app_data)
        app_entry["versions"] = {}

        # 复制应用级静态文件：logo.png、README*.md
        app_out_dir = OUTPUT_DOWNLOAD / app_name
        app_out_dir.mkdir(parents=True, exist_ok=True)
        for src_file in app_dir.iterdir():
            if src_file.is_file() and (
                src_file.name == "logo.png"
                or (src_file.name.startswith("README") and src_file.suffix.lower() == ".md")
            ):
                shutil.copy2(src_file, app_out_dir / src_file.name)
                print(f"  [复制] {app_name}/{src_file.name}")

        # 遍历每个版本目录
        for version_dir in sorted(app_dir.iterdir()):
            if not version_dir.is_dir():
                continue

            version = version_dir.name
            version_data_file = version_dir / "data.yml"
            if not version_data_file.exists():
                version_data_file = version_dir / "data.yaml"

            version_data = {}
            if version_data_file.exists():
                version_data = filter_chinese_content(load_yaml(version_data_file))
                print(f"  [版本] {app_name}/{version}")

            version_entry = dict(version_data)

            # 打包 zip
            build_version_zip(app_name, version, version_dir)

            # 扫描 compose 文件中的插值变量，并合并到 formFields
            compose_vars = extract_compose_variables(version_dir)
            if compose_vars:
                existing_fields = version_entry.get("formFields") or []
                merged_fields = merge_form_fields(existing_fields, compose_vars)
                auto_keys = [
                    f["envKey"] for f in merged_fields
                    if isinstance(f, dict) and f.get("auto") and f.get("envKey")
                ]
                if auto_keys:
                    print(f"  [变量] 自动补充: {', '.join(auto_keys)}")
                version_entry["formFields"] = merged_fields

            app_entry["versions"][version] = version_entry

        index["apps"][app_name] = app_entry

    # 合并所有层级的 additionalProperties
    index = merge_additional_properties(index)
    return index


def main():
    print("=" * 50)
    print("开始构建应用市场...")
    print("=" * 50)

    # 清理并重建 download 目录
    if OUTPUT_DOWNLOAD.exists():
        shutil.rmtree(OUTPUT_DOWNLOAD)
    OUTPUT_DOWNLOAD.mkdir(parents=True)

    # 使用临时目录下载并解压
    with tempfile.TemporaryDirectory() as tmp:
        tmp_path = Path(tmp)
        zip_path = tmp_path / "applist.zip"

        # 下载
        download_source(SOURCE_URL, zip_path)

        # 解压
        source_dir = extract_source(zip_path, tmp_path / "src")
        print(f"[源码] {source_dir}")

        # 构建 index
        index = build_index(source_dir)

    # 写出 index.json
    with open(OUTPUT_INDEX, "w", encoding="utf-8") as f:
        json.dump(index, f, ensure_ascii=False, indent=2)

    print("=" * 50)
    print("完成！")
    print(f"  index.json -> {OUTPUT_INDEX.relative_to(BASE_DIR)}")
    print(f"  下载包目录  -> {OUTPUT_DOWNLOAD.relative_to(BASE_DIR)}/")
    print("=" * 50)


if __name__ == "__main__":
    main()
