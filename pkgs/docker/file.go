package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/rehiy/libgo/logman"
)

// ContainerFileInfo 容器内文件信息
type ContainerFileInfo struct {
	Name    string `json:"name"`    // 文件名
	Size    int64  `json:"size"`    // 文件大小（字节）
	Mode    string `json:"mode"`    // 字符串权限，如 "-rw-r--r--"
	ModTime int64  `json:"modTime"` // 修改时间（Unix 时间戳）
	IsDir   bool   `json:"isDir"`   // 是否为目录
	IsLink  bool   `json:"isLink"`  // 是否为符号链接
}

// ContainerFileListResult 目录列表结果
type ContainerFileListResult struct {
	Path  string              `json:"path"`  // 当前目录路径
	Files []ContainerFileInfo `json:"files"` // 目录下的文件/子目录列表
}

// ContainerFileList 列出容器内目录内容，通过 exec ls 实现（兼容所有镜像）
func (s *DockerService) ContainerFileList(ctx context.Context, containerID, dirPath string) (*ContainerFileListResult, error) {
	if dirPath == "" {
		dirPath = "/"
	}

	// 先尝试 GNU find -printf（获取结构化信息），输出含 | 分隔符
	// format: type|name|size|mode|mtime
	findScript := fmt.Sprintf(
		`find %s -maxdepth 1 -mindepth 1 -printf '%%y|%%f|%%s|%%#m|%%T@\n'`,
		shellQuote(dirPath),
	)
	out, err := s.ContainerExecRun(ctx, containerID, "/bin/sh", findScript, 10)
	// 判断是否为有效的 find -printf 输出（每行含 4 个 |）
	if err == nil && isFindOutput(out) {
		return parseFindOutput(dirPath, out), nil
	}

	// 回退：ls -la（busybox / alpine 兼容）
	lsScript := fmt.Sprintf(`ls -la %s`, shellQuote(dirPath))
	out, err = s.ContainerExecRun(ctx, containerID, "/bin/sh", lsScript, 10)
	if err != nil && out == "" {
		return nil, fmt.Errorf("列出目录失败: %w", err)
	}
	return parseLsOutput(dirPath, out), nil
}

// isFindOutput 检查输出是否为 find -printf 格式（每行含 4 个 | 分隔符）
func isFindOutput(output string) bool {
	for _, line := range strings.SplitN(strings.TrimSpace(output), "\n", 3) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return strings.Count(line, "|") >= 4
	}
	return true // 空目录也视为有效
}

// ContainerFileDownload 从容器内下载文件，写入 dst
func (s *DockerService) ContainerFileDownload(ctx context.Context, containerID, filePath string, dst io.Writer) error {
	rc, _, err := s.client.CopyFromContainer(ctx, containerID, filePath)
	if err != nil {
		logman.Error("CopyFromContainer failed", "container", containerID, "path", filePath, "error", err)
		return fmt.Errorf("下载失败: %w", err)
	}
	defer rc.Close()

	tr := tar.NewReader(rc)
	_, err = tr.Next()
	if err != nil {
		return fmt.Errorf("读取 tar 头失败: %w", err)
	}
	if _, err := io.Copy(dst, tr); err != nil {
		return fmt.Errorf("读取文件内容失败: %w", err)
	}
	return nil
}

// ContainerFileUpload 上传单个文件到容器内指定目录。
// fileName 可含子目录（如 "subdir/file.txt"），此时会自动创建父目录。
func (s *DockerService) ContainerFileUpload(ctx context.Context, containerID, destDir, fileName string, content io.Reader) error {
	data, err := io.ReadAll(content)
	if err != nil {
		return fmt.Errorf("读取上传内容失败: %w", err)
	}

	// 若 fileName 含子目录，预先在容器内创建对应目录
	if dir := path.Dir(fileName); dir != "" && dir != "." && dir != "/" {
		if mkErr := s.ContainerFileMkdir(ctx, containerID, path.Join(destDir, dir)); mkErr != nil {
			logman.Warn("ContainerFileUpload mkdir failed", "dir", dir, "error", mkErr)
		}
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name:     fileName,
		Size:     int64(len(data)),
		Mode:     0644,
		ModTime:  time.Now(),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("写 tar 头失败: %w", err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("写 tar 内容失败: %w", err)
	}
	tw.Close()

	err = s.client.CopyToContainer(ctx, containerID, destDir, &buf, container.CopyToContainerOptions{})
	if err != nil {
		logman.Error("CopyToContainer failed", "container", containerID, "dest", destDir, "error", err)
		return fmt.Errorf("上传失败: %w", err)
	}
	return nil
}

// ContainerFileRemove 删除容器内文件或目录
func (s *DockerService) ContainerFileRemove(ctx context.Context, containerID, targetPath string, recursive bool) error {
	cmd := fmt.Sprintf("rm -f %s", shellQuote(targetPath))
	if recursive {
		cmd = fmt.Sprintf("rm -rf %s", shellQuote(targetPath))
	}
	_, err := s.ContainerExecRun(ctx, containerID, "/bin/sh", cmd, 30)
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}
	return nil
}

// ContainerFileMkdir 在容器内创建目录
func (s *DockerService) ContainerFileMkdir(ctx context.Context, containerID, dirPath string) error {
	cmd := fmt.Sprintf("mkdir -p %s", shellQuote(dirPath))
	_, err := s.ContainerExecRun(ctx, containerID, "/bin/sh", cmd, 10)
	if err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	return nil
}

// ContainerFileRename 重命名/移动容器内文件
func (s *DockerService) ContainerFileRename(ctx context.Context, containerID, oldPath, newPath string) error {
	cmd := fmt.Sprintf("mv %s %s", shellQuote(oldPath), shellQuote(newPath))
	_, err := s.ContainerExecRun(ctx, containerID, "/bin/sh", cmd, 10)
	if err != nil {
		return fmt.Errorf("重命名失败: %w", err)
	}
	return nil
}

// ContainerFileRead 读取容器内文件文本内容
func (s *DockerService) ContainerFileRead(ctx context.Context, containerID, filePath string) (string, error) {
	var buf bytes.Buffer
	if err := s.ContainerFileDownload(ctx, containerID, filePath, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ContainerFileWrite 写入文本内容到容器内文件
func (s *DockerService) ContainerFileWrite(ctx context.Context, containerID, filePath, content string) error {
	dir := path.Dir(filePath)
	name := path.Base(filePath)
	return s.ContainerFileUpload(ctx, containerID, dir, name, strings.NewReader(content))
}

// ContainerFileChmod 修改容器内文件权限
func (s *DockerService) ContainerFileChmod(ctx context.Context, containerID, targetPath, mode string) error {
	cmd := fmt.Sprintf("chmod %s %s", shellQuote(mode), shellQuote(targetPath))
	_, err := s.ContainerExecRun(ctx, containerID, "/bin/sh", cmd, 10)
	if err != nil {
		return fmt.Errorf("修改权限失败: %w", err)
	}
	return nil
}

// shellQuote 对路径进行简单单引号转义，防止路径注入
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// parseFindOutput 解析 find -printf '%y|%f|%s|%#m|%T@\n' 输出
func parseFindOutput(dirPath, output string) *ContainerFileListResult {
	result := &ContainerFileListResult{Path: dirPath, Files: []ContainerFileInfo{}}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}
		ftype, fname, fsize, fmode, fmtime := parts[0], parts[1], parts[2], parts[3], parts[4]
		if fname == "" || fname == "." || fname == ".." {
			continue
		}

		var size int64
		fmt.Sscanf(fsize, "%d", &size)
		var mtime float64
		fmt.Sscanf(fmtime, "%f", &mtime)

		isDir := ftype == "d"
		isLink := ftype == "l"
		mode := octalToStr(fmode, isDir)

		result.Files = append(result.Files, ContainerFileInfo{
			Name:    fname,
			Size:    size,
			Mode:    mode,
			ModTime: int64(mtime),
			IsDir:   isDir,
			IsLink:  isLink,
		})
	}
	return result
}

// parseLsOutput 简单解析 ls -la 输出（回退方案）
func parseLsOutput(dirPath, output string) *ContainerFileListResult {
	result := &ContainerFileListResult{Path: dirPath, Files: []ContainerFileInfo{}}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		modeStr := fields[0]
		name := strings.Join(fields[8:], " ")
		// 去掉 -> 链接目标
		if idx := strings.Index(name, " -> "); idx >= 0 {
			name = name[:idx]
		}
		if name == "." || name == ".." {
			continue
		}
		var size int64
		fmt.Sscanf(fields[4], "%d", &size)

		isDir := len(modeStr) > 0 && modeStr[0] == 'd'
		isLink := len(modeStr) > 0 && modeStr[0] == 'l'
		// ls -la 已输出字符串权限，转换为标准格式（确保 modeOctal 可用）
		normalizedMode := normalizeLsMode(modeStr, isDir)

		result.Files = append(result.Files, ContainerFileInfo{
			Name:    name,
			Size:    size,
			Mode:    normalizedMode,
			ModTime: 0,
			IsDir:   isDir,
			IsLink:  isLink,
		})
	}
	return result
}

// normalizeLsMode 将 ls -la 输出的权限字符串标准化为固定 10 位格式（如 "-rw-r--r--"）
// ls 输出已经是 10 位，但 ACL 等场景可能有 + 后缀，统一截取前 10 位
func normalizeLsMode(mode string, isDir bool) string {
	if len(mode) >= 10 {
		return mode[:10]
	}
	if isDir {
		return "drwxr-xr-x"
	}
	return "-rw-r--r--"
}

// octalToStr 将八进制权限字符串（如 0755）转为 rwxr-xr-x 格式
func octalToStr(octal string, isDir bool) string {
	octal = strings.TrimPrefix(octal, "0")
	if len(octal) < 3 {
		if isDir {
			return "drwxr-xr-x"
		}
		return "-rw-r--r--"
	}
	prefix := "-"
	if isDir {
		prefix = "d"
	}
	toRwx := func(c byte) string {
		v := int(c - '0')
		r, w, x := "-", "-", "-"
		if v&4 != 0 {
			r = "r"
		}
		if v&2 != 0 {
			w = "w"
		}
		if v&1 != 0 {
			x = "x"
		}
		return r + w + x
	}
	o := octal
	if len(o) == 4 {
		o = o[1:] // 去掉特殊位
	}
	if len(o) < 3 {
		o = "644"
	}
	return prefix + toRwx(o[0]) + toRwx(o[1]) + toRwx(o[2])
}
