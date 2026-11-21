package docker

import (
	"archive/tar"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/docker/docker/api/types/container"
	"github.com/rehiy/libgo/command"
	"github.com/rehiy/libgo/filer"
	"github.com/rehiy/libgo/logman"

	"isrvd/pkgs/helper"
)

// ContainerFileInfo 容器内文件信息
type ContainerFileInfo struct {
	Name       string `json:"name"`                 // 文件名
	Size       int64  `json:"size"`                 // 文件大小（字节）
	Mode       string `json:"mode"`                 // 字符串权限，如 "-rw-r--r--"
	ModTime    int64  `json:"modTime"`              // 修改时间（Unix 时间戳）
	IsDir      bool   `json:"isDir"`                // 是否为目录
	IsLink     bool   `json:"isLink"`               // 是否为符号链接
	LinkTarget string `json:"linkTarget,omitempty"` // 符号链接目标
}

// ContainerFileListResult 目录列表结果
type ContainerFileListResult struct {
	Path  string              `json:"path"`  // 当前目录路径
	Files []ContainerFileInfo `json:"files"` // 目录下的文件/子目录列表
}

// ContainerFileList 使用 NUL 分隔字段，保留任意可用 UTF-8 表示的文件名。
func (s *DockerService) ContainerFileList(ctx context.Context, containerID, dirPath string) (*ContainerFileListResult, error) {
	if dirPath == "" {
		dirPath = "/"
	}
	listPath := command.POSIXQuote(containerDirListPath(dirPath))
	findScript := fmt.Sprintf(`find %s -maxdepth 1 -mindepth 1 -printf '%%y\0%%Y\0%%f\0%%s\0%%#m\0%%T@\0%%l\0'`, listPath)
	out, truncated, err := s.containerExecRun(ctx, containerID, "/bin/sh", findScript, 10)
	if _, exited := errors.AsType[commandExitError](err); exited && ctx.Err() == nil && !truncated {
		// BusyBox find 不支持 -printf。用 glob、stat 及 readlink 输出相同格式。
		script := "dir=" + listPath + "\n" + `
LC_ALL=C
export LC_ALL
[ -d "$dir" ] && [ -r "$dir" ] && [ -x "$dir" ] || exit 1
for entry in "$dir"/* "$dir"/.[!.]* "$dir"/..?*; do
    [ -e "$entry" ] || [ -L "$entry" ] || continue
    kind=f
    if [ -L "$entry" ]; then kind=l; elif [ -d "$entry" ]; then kind=d; fi
    metadata=$(stat -c '%s %a %Y' "$entry") || exit 1
    set -- $metadata
    [ "$#" -eq 3 ] || exit 1
    printf '%s\0%s\0%s\0%s\0%s\0%s\0' "$kind" "$kind" "${entry##*/}" "$1" "$2" "$3"
    if [ "$kind" = l ]; then readlink -n "$entry" || exit 1; fi
    printf '\0'
done`
		out, truncated, err = s.containerExecRun(ctx, containerID, "/bin/sh", script, 10)
	}
	if truncated {
		return nil, errors.New("容器目录列表超过输出上限")
	}
	if err != nil {
		return nil, fmt.Errorf("列出目录失败: %w", err)
	}
	result, err := parseDirectoryRecords(dirPath, out)
	if err != nil {
		return nil, err
	}
	s.resolveContainerLinkTargets(ctx, containerID, result)
	return result, ctx.Err()
}

func (s *DockerService) resolveContainerLinkTargets(ctx context.Context, containerID string, result *ContainerFileListResult) {
	if result == nil {
		return
	}
	for i := range result.Files {
		file := &result.Files[i]
		if file.LinkTarget == "" {
			continue
		}
		file.IsLink = true
		stat, err := s.client.ContainerStatPath(ctx, containerID, containerLinkTargetPath(result.Path, file.LinkTarget))
		if err == nil && stat.Mode.IsDir() {
			file.IsDir = true
			file.Mode = "d" + file.Mode[1:]
		}
	}
}

// ContainerFileOpen 打开容器归档中的普通文件并返回其长度。
func (s *DockerService) ContainerFileOpen(ctx context.Context, containerID, filePath string) (io.ReadCloser, int64, error) {
	body, _, err := s.client.CopyFromContainer(ctx, containerID, filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("下载失败: %w", err)
	}
	tr := tar.NewReader(body)
	hdr, err := tr.Next()
	if err == nil && hdr.Typeflag != tar.TypeReg {
		err = os.ErrInvalid
	}
	if err != nil {
		body.Close()
		return nil, 0, fmt.Errorf("读取 tar 头失败: %w", err)
	}
	return struct {
		io.Reader
		io.Closer
	}{tr, body}, hdr.Size, nil
}

// ContainerFileUpload 将已知长度的文件流式打包到容器。
func (s *DockerService) ContainerFileUpload(ctx context.Context, containerID, destDir, fileName string, content io.Reader, size int64) error {
	if size < 0 {
		return os.ErrInvalid
	}
	if dir := path.Dir(fileName); dir != "." && dir != "/" {
		if err := s.ContainerFileMkdir(ctx, containerID, path.Join(destDir, dir)); err != nil {
			return err
		}
	}

	reader, writer := io.Pipe()
	packed := make(chan error, 1)
	go func() {
		tarWriter := tar.NewWriter(writer)
		err := tarWriter.WriteHeader(&tar.Header{Name: fileName, Size: size, Mode: 0644, ModTime: time.Now()})
		if err == nil {
			src := helper.ContextReader{Context: ctx, Reader: content}
			_, err = io.CopyN(tarWriter, src, size)
			if err == nil {
				var extra [1]byte
				if n, readErr := io.ReadFull(src, extra[:]); n != 0 || readErr != io.EOF {
					err = io.ErrShortWrite
					if n == 0 && readErr != nil {
						err = readErr
					}
				}
			}
		}
		if err == nil {
			err = tarWriter.Close()
		}
		_ = writer.CloseWithError(err)
		packed <- err
	}()
	err := s.client.CopyToContainer(ctx, containerID, destDir, reader, container.CopyToContainerOptions{})
	_ = reader.Close()
	if err := errors.Join(err, <-packed); err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}
	return nil
}

// ContainerFileRemove 删除容器内文件或目录
func (s *DockerService) ContainerFileRemove(ctx context.Context, containerID, targetPath string, recursive bool) error {
	cmd := fmt.Sprintf("rm -f %s", command.POSIXQuote(targetPath))
	if recursive {
		cmd = fmt.Sprintf("rm -rf %s", command.POSIXQuote(targetPath))
	}
	_, err := s.ContainerExecRun(ctx, containerID, "/bin/sh", cmd, 30)
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}
	return nil
}

// ContainerFileMkdir 在容器内创建目录
func (s *DockerService) ContainerFileMkdir(ctx context.Context, containerID, dirPath string) error {
	cmd := fmt.Sprintf("mkdir -p %s", command.POSIXQuote(dirPath))
	_, err := s.ContainerExecRun(ctx, containerID, "/bin/sh", cmd, 10)
	if err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	return nil
}

// ContainerFileRename 重命名/移动容器内文件
func (s *DockerService) ContainerFileRename(ctx context.Context, containerID, oldPath, newPath string) error {
	cmd := fmt.Sprintf("mv %s %s", command.POSIXQuote(oldPath), command.POSIXQuote(newPath))
	_, err := s.ContainerExecRun(ctx, containerID, "/bin/sh", cmd, 10)
	if err != nil {
		return fmt.Errorf("重命名失败: %w", err)
	}
	return nil
}

// ContainerFileRead 读取用于在线编辑的容器文件内容，超大文件应使用流式下载接口。
func (s *DockerService) ContainerFileRead(ctx context.Context, containerID, filePath string) (string, error) {
	buf := &helper.EditBuffer{}
	src, size, err := s.ContainerFileOpen(ctx, containerID, filePath)
	if err != nil {
		return "", err
	}
	defer src.Close()
	if size > helper.MaxEditableFileBytes {
		return "", helper.ErrTooLarge
	}
	_, err = io.Copy(buf, src)
	if buf.Exceeded() {
		return "", helper.ErrTooLarge
	}
	if err != nil {
		return "", err
	}
	if !filer.IsTextBytes(buf.Bytes()) {
		return "", helper.ErrBinary
	}
	return strings.ToValidUTF8(buf.String(), "\uFFFD"), nil
}

// ContainerFileWrite 写入文本内容到容器内文件
func (s *DockerService) ContainerFileWrite(ctx context.Context, containerID, filePath, content string) error {
	dir := path.Dir(filePath)
	name := path.Base(filePath)
	return s.ContainerFileUpload(ctx, containerID, dir, name, strings.NewReader(content), int64(len(content)))
}

// ContainerFileCreateOnly 写入临时文件，再以硬链接原子发布，避免容器内同名写入。
func (s *DockerService) ContainerFileCreateOnly(ctx context.Context, containerID, filePath, content string) error {
	staged := path.Join(path.Dir(filePath), ".isrvd-create-"+rand.Text())
	quotedStage := command.POSIXQuote(staged)
	if _, err := s.ContainerExecRun(ctx, containerID, "/bin/sh", "mkdir -m 700 -- "+quotedStage, 10); err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if _, err := s.ContainerExecRun(cleanup, containerID, "/bin/sh", "rm -rf -- "+quotedStage, 5); err != nil {
			logman.Warn("Remove staged container file failed", "error", err)
		}
	}()
	if err := s.ContainerFileUpload(ctx, containerID, staged, "content", strings.NewReader(content), int64(len(content))); err != nil {
		return err
	}
	const conflict = 73
	script := "destination=" + command.POSIXQuote(filePath) + "\n" +
		"ln -T -- " + command.POSIXQuote(path.Join(staged, "content")) + " \"$destination\" && exit 0\n" +
		fmt.Sprintf("if [ -e \"$destination\" ] || [ -L \"$destination\" ]; then exit %d; fi\nexit 1", conflict)
	_, err := s.ContainerExecRun(ctx, containerID, "/bin/sh", script, 10)
	if status, ok := errors.AsType[commandExitError](err); ok && status == conflict {
		return os.ErrExist
	}
	return err
}

// ContainerFileChmod 修改容器内文件权限
func (s *DockerService) ContainerFileChmod(ctx context.Context, containerID, targetPath, mode string) error {
	cmd := fmt.Sprintf("chmod %s %s", command.POSIXQuote(mode), command.POSIXQuote(targetPath))
	_, err := s.ContainerExecRun(ctx, containerID, "/bin/sh", cmd, 10)
	if err != nil {
		return fmt.Errorf("修改权限失败: %w", err)
	}
	return nil
}

// ─── 辅助函数 ───

// containerLinkTargetPath 返回软链接目标在容器内的绝对路径。
func containerLinkTargetPath(parentPath, targetPath string) string {
	if targetPath == "" {
		return path.Clean(parentPath)
	}
	if path.IsAbs(targetPath) {
		return path.Clean(targetPath)
	}
	return path.Clean(path.Join(parentPath, targetPath))
}

// containerDirListPath 返回用于列目录的容器内路径。
// 追加 /. 可让 find 展开命令行参数中的目录软链接，例如 /sbin -> /usr/sbin。
func containerDirListPath(dirPath string) string {
	if dirPath == "" {
		return "/"
	}
	cleaned := path.Clean(dirPath)
	if cleaned == "." {
		return "."
	}
	if cleaned == "/" {
		return "/"
	}
	return strings.TrimRight(cleaned, "/") + "/."
}

func parseDirectoryRecords(dirPath, output string) (*ContainerFileListResult, error) {
	result := &ContainerFileListResult{Path: dirPath, Files: []ContainerFileInfo{}}
	if output == "" {
		return result, nil
	}
	fields := strings.Split(output, "\x00")
	if !utf8.ValidString(output) || fields[len(fields)-1] != "" || (len(fields)-1)%7 != 0 {
		return nil, errors.New("容器目录记录格式无效")
	}
	for i := 0; i < len(fields)-1; i += 7 {
		kind, target, name, link := fields[i], fields[i+1], fields[i+2], fields[i+6]
		size, sizeErr := strconv.ParseInt(fields[i+3], 10, 64)
		perm, permErr := strconv.ParseUint(fields[i+4], 8, 16)
		mtime, timeErr := strconv.ParseFloat(fields[i+5], 64)
		if sizeErr != nil || permErr != nil || timeErr != nil || name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
			return nil, errors.New("容器目录记录格式无效")
		}
		isLink := kind == "l"
		isDir := kind == "d" || isLink && target == "d"
		mode := os.FileMode(perm) & os.ModePerm
		if isDir {
			mode |= os.ModeDir
		}
		result.Files = append(result.Files, ContainerFileInfo{
			Name: name, Size: size, Mode: mode.String(),
			ModTime: int64(mtime), IsDir: isDir, IsLink: isLink, LinkTarget: link,
		})
	}
	return result, nil
}
