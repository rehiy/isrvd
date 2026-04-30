package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
)

// composeFileCandidates 常见的 compose 文件名（按优先级）
var composeFileCandidates = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yaml",
	"docker-compose.yml",
}

// LoadOptions 加载配置
type LoadOptions struct {
	// WorkingDir compose 文件所在目录（用于解析相对路径和 .env）
	WorkingDir string
	// ConfigFiles 指定要加载的 compose 文件绝对路径；为空则在 WorkingDir 下自动查找
	ConfigFiles []string
	// ProjectName 项目名（影响生成的网络/容器默认前缀）
	ProjectName string
	// Environment 额外的环境变量（会与 .env 合并，覆盖 .env）
	Environment map[string]string
}

// LoadProject 使用 compose-go 官方加载器解析 compose 文件，
// 自动处理 .env、变量插值、schema 校验等一致性检查
func LoadProject(ctx context.Context, opts LoadOptions) (*types.Project, error) {
	if opts.WorkingDir == "" {
		return nil, fmt.Errorf("缺少工作目录")
	}

	configFiles := opts.ConfigFiles
	if len(configFiles) == 0 {
		found, err := LocateComposeFile(opts.WorkingDir)
		if err != nil {
			return nil, err
		}
		configFiles = []string{found}
	}

	files := make([]types.ConfigFile, 0, len(configFiles))
	for _, p := range configFiles {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("读取 compose 文件失败: %w", err)
		}
		files = append(files, types.ConfigFile{
			Filename: p,
			Content:  data,
		})
	}

	// 合并 host 环境变量 + 用户自定义（用户优先）
	env := loadHostEnv()
	for k, v := range opts.Environment {
		env[k] = v
	}

	details := types.ConfigDetails{
		WorkingDir:  opts.WorkingDir,
		ConfigFiles: files,
		Environment: env,
	}

	projectName := opts.ProjectName
	if projectName == "" {
		projectName = filepath.Base(opts.WorkingDir)
	}
	projectName = loader.NormalizeProjectName(projectName)

	project, err := loader.LoadWithContext(ctx, details, func(o *loader.Options) {
		o.SetProjectName(projectName, true)
		o.SkipConsistencyCheck = false
		o.SkipValidation = false
		o.ResolvePaths = true
	})
	if err != nil {
		return nil, fmt.Errorf("加载 compose 失败: %w", err)
	}

	return project, nil
}

// LoadProjectFromContent 从 yaml 文本直接加载 compose 项目
// 用于「手动粘贴 compose 内容」场景，临时写入内存 buffer 即可
func LoadProjectFromContent(ctx context.Context, content string, projectName string) (*types.Project, error) {
	if content == "" {
		return nil, fmt.Errorf("compose 内容为空")
	}

	details := types.ConfigDetails{
		WorkingDir: ".",
		ConfigFiles: []types.ConfigFile{
			{
				Filename: "compose.yml",
				Content:  []byte(content),
			},
		},
		Environment: loadHostEnv(),
	}

	if projectName == "" {
		projectName = "isrvd"
	}
	projectName = loader.NormalizeProjectName(projectName)

	project, err := loader.LoadWithContext(ctx, details, func(o *loader.Options) {
		o.SetProjectName(projectName, true)
		o.SkipConsistencyCheck = false
		o.SkipValidation = false
		o.ResolvePaths = false
	})
	if err != nil {
		return nil, fmt.Errorf("加载 compose 失败: %w", err)
	}

	return project, nil
}

// loadHostEnv 将当前进程的环境变量加载为 map，用于 compose 变量插值
func loadHostEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				env[e[:i]] = e[i+1:]
				break
			}
		}
	}
	return env
}

// LocateComposeFile 在指定目录下查找 compose 文件
// 查找顺序：根目录 → 一级子目录（处理 zip 解压后外层多一层目录的情况）
func LocateComposeFile(dir string) (string, error) {
	for _, name := range composeFileCandidates {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("读取目录失败: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := filepath.Join(dir, e.Name())
		for _, name := range composeFileCandidates {
			p := filepath.Join(sub, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("目录中未找到 compose 文件")
}
