// Package compose 提供基于 compose-spec 官方规范的 compose 文件加载、变量插值、
// YAML 序列化以及与 pkgs/docker、pkgs/swarm 数据结构的双向转换能力。
//
// 本包是解析/转换层，附带 compose.yml 与 .env 的落盘能力（目录与内容均由调用方传入），
// 不直接操作底层容器/网络/卷的 CRUD —
// 具体的部署、网络确保、镜像拉取等编排能力由上层 service/compose 完成。
package compose

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/dotenv"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/goccy/go-yaml"
)

// ProjectToYAML 使用 compose-go 官方序列化器输出 YAML（不含文件头，由调用方按需添加）
func ProjectToYAML(project *types.Project) ([]byte, error) {
	if project == nil {
		return nil, fmt.Errorf("project 为空")
	}
	data, err := project.MarshalYAML()
	if err != nil {
		return nil, fmt.Errorf("序列化 compose 失败: %w", err)
	}
	return data, nil
}

// LoadProject 加载项目目录中由 isrvd 管理的 compose.yml。
func LoadProject(ctx context.Context, workingDir, projectName string) (*types.Project, error) {
	if workingDir == "" {
		return nil, fmt.Errorf("缺少工作目录")
	}
	content, err := os.ReadFile(filepath.Join(workingDir, ComposeFileName))
	if err != nil {
		return nil, fmt.Errorf("读取 compose 文件失败: %w", err)
	}
	if projectName == "" {
		projectName = filepath.Base(workingDir)
	}
	return LoadProjectFromContentInDir(ctx, string(content), workingDir, projectName, nil)
}

// LoadProjectFromContentInDir 从 yaml 文本加载 compose 项目，并以 workingDir 作为项目目录解析相对路径。
// workingDir 为空时回退到进程当前目录（与上游行为一致）。
// envContent 为 nil 时读取 workingDir/.env；非 nil 时替代该文件，空字符串表示清空。
func LoadProjectFromContentInDir(ctx context.Context, content, workingDir, projectName string, envContent *string) (*types.Project, error) {
	return loadProjectContent(ctx, content, workingDir, projectName, true, envContent, false)
}

// loadProjectContent 从 yaml 文本加载 compose 项目，并合并指定的 .env 内容到插值环境。
// envContent 为 nil 时读取 workingDir/.env；非 nil 时替代该文件，空字符串表示清空。
// skipServiceEnvironmentResolution 为 true 时不合并 services.environment/env_file。
func loadProjectContent(ctx context.Context, content, workingDir, projectName string, resolvePaths bool, envContent *string, skipServiceEnvironmentResolution bool) (*types.Project, error) {
	if content == "" {
		return nil, fmt.Errorf("compose 内容为空")
	}
	workingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, err
	}

	env, err := projectEnvironmentWithEnvFile(ctx, nil, envContent, workingDir)
	if err != nil {
		return nil, err
	}

	details := types.ConfigDetails{
		WorkingDir: workingDir,
		ConfigFiles: []types.ConfigFile{
			{
				Filename: filepath.Join(workingDir, ComposeFileName),
				Content:  []byte(content),
			},
		},
		Environment: env,
	}

	project, err := loadProjectWithOptions(ctx, details, projectName, resolvePaths, projectName != "", func(o *loader.Options) {
		o.SkipResolveEnvironment = skipServiceEnvironmentResolution || envContent != nil
	})
	if err != nil || envContent == nil || skipServiceEnvironmentResolution {
		return project, err
	}
	return resolveProjectEnv(ctx, project, *envContent)
}

// resolveProjectEnv 将项目 .env 引用暂时指向本次提交的内容，顺序、格式和优先级由 compose-go 解析。
func resolveProjectEnv(ctx context.Context, project *types.Project, content string) (*types.Project, error) {
	original := filepath.Join(project.WorkingDir, EnvFileName)
	if !replaceProjectEnvPath(project, original, original) {
		return project.WithServicesEnvironmentResolved(false)
	}
	staged, err := os.CreateTemp("", "isrvd-compose-env-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(staged.Name())
	_, err = staged.WriteString(content)
	if err := errors.Join(err, staged.Close()); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	replaceProjectEnvPath(project, original, staged.Name())
	resolved, err := project.WithServicesEnvironmentResolved(false)
	if err != nil {
		return nil, err
	}
	replaceProjectEnvPath(resolved, staged.Name(), original)
	return resolved, ctx.Err()
}

func replaceProjectEnvPath(project *types.Project, from, to string) bool {
	found := false
	replace := func(files []types.EnvFile) {
		for i := range files {
			if filepath.Clean(files[i].Path) == from {
				files[i].Path = to
				found = true
			}
		}
	}
	for _, service := range project.Services {
		replace(service.EnvFiles)
		for _, hook := range service.PreStart {
			replace(hook.EnvFiles)
		}
	}
	return found
}

func loadProjectWithOptions(ctx context.Context, details types.ConfigDetails, projectName string, resolvePaths bool, setProjectName bool, configure func(*loader.Options)) (*types.Project, error) {
	projectName = loader.NormalizeProjectName(projectName)
	project, err := loader.LoadWithContext(ctx, details, func(o *loader.Options) {
		if setProjectName && projectName != "" {
			o.SetProjectName(projectName, true)
		}
		o.SkipConsistencyCheck = false
		o.SkipValidation = false
		o.ResolvePaths = resolvePaths
		if configure != nil {
			configure(o)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("加载 compose 失败: %w", err)
	}
	return project, nil
}

// ProjectValidateWithoutEnvFiles 在不读取 services.env_file 的前提下校验 Compose 结构，
// 要求显式指定非空项目名。
// 用于部署落盘前预检；env_file 仅在文件就位后的正式加载阶段解析。
func ProjectValidateWithoutEnvFiles(ctx context.Context, content string, envContent *string) (*types.Project, error) {
	if content == "" {
		return nil, fmt.Errorf("compose 内容为空")
	}
	env, err := projectEnvironmentWithEnvFile(ctx, nil, envContent, "")
	if err != nil {
		return nil, err
	}
	details := types.ConfigDetails{
		WorkingDir: ".",
		ConfigFiles: []types.ConfigFile{{
			Filename: "compose.yml",
			Content:  []byte(content),
		}},
		Environment: env,
	}
	skipEnvFiles := func(o *loader.Options) {
		o.SkipResolveEnvironment = true
	}

	if !HasProjectName(content) {
		return nil, fmt.Errorf("compose 项目必须指定非空 name")
	}
	return loadProjectWithOptions(ctx, details, "", false, false, skipEnvFiles)
}

func projectEnvironment(extra map[string]string) map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if k, v, ok := strings.Cut(e, "="); ok {
			env[k] = v
		}
	}
	maps.Copy(env, extra)
	return env
}

// projectEnvironmentWithEnvFile 构建 ${VAR} 插值环境。
func projectEnvironmentWithEnvFile(ctx context.Context, extra map[string]string, envContent *string, workingDir string) (map[string]string, error) {
	env := make(map[string]string)

	// 提交了 envContent 时，它就是本次部署的完整 .env，不能继承旧文件的变量。
	if workingDir != "" && envContent == nil {
		envFilePath := filepath.Join(workingDir, EnvFileName)
		if data, err := os.ReadFile(envFilePath); err == nil {
			fileEnv, perr := parseEnvContent(string(data))
			if perr != nil {
				return nil, fmt.Errorf("解析 %s 失败: %w", envFilePath, perr)
			}
			maps.Copy(env, fileEnv)
		} else if !errors.Is(err, os.ErrNotExist) {
			slog.DebugContext(ctx, "read project .env failed", "path", envFilePath, "error", err)
		}
	}

	if envContent != nil && strings.TrimSpace(*envContent) != "" {
		fileEnv, err := parseEnvContent(*envContent)
		if err != nil {
			return nil, fmt.Errorf("解析 .env 内容失败: %w", err)
		}
		maps.Copy(env, fileEnv)
	}

	maps.Copy(env, projectEnvironment(extra))

	return env, nil
}

// parseEnvContent 解析 KEY=VALUE 形式的 .env 文本。
func parseEnvContent(content string) (map[string]string, error) {
	return dotenv.Parse(bytes.NewReader([]byte(content)))
}

// HasProjectName 检查顶层 name 是否为非空字符串。
func HasProjectName(content string) bool {
	var doc struct {
		Name *string `yaml:"name"`
	}
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return false
	}
	return doc.Name != nil && strings.TrimSpace(*doc.Name) != ""
}
