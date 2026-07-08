package swarm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerSwarm "github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rehiy/libgo/httpd"
	"github.com/rehiy/libgo/logman"

	pkgDocker "isrvd/pkgs/docker"
)

// ServiceList 获取服务列表，直接返回 Docker SDK 原始服务结构。
func (s *SwarmService) ServiceList(ctx context.Context) ([]dockerSwarm.Service, error) {
	services, err := s.client.ServiceList(ctx, dockerSwarm.ServiceListOptions{})
	if err != nil {
		logman.Error("ServiceList failed", "error", err)
		return nil, err
	}
	return services, nil
}

// ServiceAction 服务操作（scale/remove/force-update）
func (s *SwarmService) ServiceAction(ctx context.Context, id, action string, replicas *uint64) error {
	if action == "remove" {
		if err := s.client.ServiceRemove(ctx, id); err != nil {
			logman.Error("ServiceRemove failed", "id", id, "error", err)
			return err
		}
		return nil
	}

	if action == "scale" && replicas != nil {
		svc, _, err := s.client.ServiceInspectWithRaw(ctx, id, dockerSwarm.ServiceInspectOptions{InsertDefaults: true})
		if err != nil {
			logman.Error("ServiceInspect failed", "id", id, "error", err)
			return err
		}
		if svc.Spec.Mode.Replicated == nil {
			return fmt.Errorf("仅 replicated 模式服务支持 scale")
		}
		svc.Spec.Mode.Replicated.Replicas = replicas
		resp, err := s.client.ServiceUpdate(ctx, id, svc.Version, svc.Spec, s.serviceUpdateOptions(svc.Spec))
		if err != nil {
			logman.Error("ServiceScale failed", "id", id, "replicas", *replicas, "error", err)
			return err
		}
		for _, warning := range resp.Warnings {
			logman.Warn("ServiceScale warning", "id", id, "warning", warning)
		}
		return nil
	}

	if action == "force-update" {
		return s.ServiceForceUpdate(ctx, id)
	}

	return fmt.Errorf("不支持的操作: %s", action)
}

// ServiceCreate 创建服务，直接接收 Docker SDK 原始 ServiceSpec。
func (s *SwarmService) ServiceCreate(ctx context.Context, spec dockerSwarm.ServiceSpec) (string, error) {
	resp, err := s.client.ServiceCreate(ctx, spec, s.serviceCreateOptions(spec))
	if err != nil {
		logman.Error("ServiceCreate failed", "error", err)
		return "", err
	}
	for _, warning := range resp.Warnings {
		logman.Warn("ServiceCreate warning", "service", spec.Name, "warning", warning)
	}
	return resp.ID, nil
}

// ServiceForceUpdate 强制重新部署服务
func (s *SwarmService) ServiceForceUpdate(ctx context.Context, id string) error {
	svc, _, err := s.client.ServiceInspectWithRaw(ctx, id, dockerSwarm.ServiceInspectOptions{InsertDefaults: true})
	if err != nil {
		logman.Error("ServiceInspect failed", "id", id, "error", err)
		return err
	}

	svc.Spec.TaskTemplate.ForceUpdate++

	resp, err := s.client.ServiceUpdate(ctx, id, svc.Version, svc.Spec, s.serviceUpdateOptions(svc.Spec))
	if err != nil {
		logman.Error("ServiceForceUpdate failed", "error", err)
		return err
	}
	for _, warning := range resp.Warnings {
		logman.Warn("ServiceForceUpdate warning", "id", id, "warning", warning)
	}
	return nil
}

// ServiceLogs 获取服务日志
func (s *SwarmService) ServiceLogs(ctx context.Context, serviceID, tail string) ([]string, error) {
	if tail == "" {
		tail = "100"
	}

	reader, err := s.client.ServiceLogs(ctx, serviceID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	})
	if err != nil {
		logman.Error("ServiceLogs failed", "error", err)
		return nil, err
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return pkgDocker.ParseDockerLogs(raw), nil
}

// ServiceLogsStream 实时转发服务日志到 writer。
// writer 可选实现 httpd.Writer 以区分 error 事件与普通 data 事件。
func (s *SwarmService) ServiceLogsStream(ctx context.Context, w io.Writer, serviceID, tail string) {
	if tail == "" {
		tail = "100"
	}

	writeError := func(msg string) {
		if sw, ok := w.(httpd.Writer); ok {
			_ = sw.WriteEvent("error", msg)
		} else {
			_, _ = w.Write([]byte("[" + msg + "]\n"))
		}
	}

	reader, err := s.client.ServiceLogs(ctx, serviceID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Follow:     true,
		Timestamps: true,
	})
	if err != nil {
		logman.Error("Start service logs stream failed", "id", serviceID, "error", err)
		writeError("获取服务日志失败: " + err.Error())
		return
	}
	defer reader.Close()

	// 心跳 ticker，保持 SSE 连接活跃
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	// Swarm 服务日志为多路复用流（无 TTY），使用 stdcopy 解帧
	errCh := make(chan error, 1)
	go func() {
		_, copyErr := stdcopy.StdCopy(w, w, reader)
		errCh <- copyErr
	}()

	for {
		select {
		case err := <-errCh:
			if err != nil && ctx.Err() == nil && !errors.Is(err, io.EOF) {
				logman.Warn("Service logs stream stopped with error", "id", serviceID, "error", err)
			}
			return
		case <-ctx.Done():
			logman.Info("Service logs stream cancelled by context", "id", serviceID)
			return
		case <-heartbeat.C:
			if sw, ok := w.(httpd.Writer); ok {
				if err := sw.WriteEvent("heartbeat", "ping"); err != nil {
					logman.Warn("Failed to send heartbeat", "id", serviceID, "error", err)
				}
			}
		}
	}
}

// ServiceInspect 获取服务详情，直接返回 Docker SDK 原始服务结构。
func (s *SwarmService) ServiceInspect(ctx context.Context, id string) (dockerSwarm.Service, error) {
	return s.ServiceInspectRaw(ctx, id)
}

// ServiceRunningTasksMap 一次性统计所有服务运行中的任务数。
func (s *SwarmService) ServiceRunningTasksMap(ctx context.Context) map[string]int {
	tasks, err := s.client.TaskList(ctx, dockerSwarm.TaskListOptions{})
	if err != nil {
		logman.Warn("TaskList failed in ServiceRunningTasksMap", "error", err)
		return map[string]int{}
	}
	runningMap := map[string]int{}
	for _, t := range tasks {
		if t.Status.State == dockerSwarm.TaskStateRunning {
			runningMap[t.ServiceID]++
		}
	}
	return runningMap
}

// ServiceRunningTasks 统计服务运行中的任务数，供需要附加运行态信息的调用方使用。
func (s *SwarmService) ServiceRunningTasks(ctx context.Context, serviceID string) int {
	f := filters.NewArgs()
	f.Add("service", serviceID)
	tasks, _ := s.client.TaskList(ctx, dockerSwarm.TaskListOptions{Filters: f})
	runningTasks := 0
	for _, t := range tasks {
		if t.Status.State == dockerSwarm.TaskStateRunning {
			runningTasks++
		}
	}
	return runningTasks
}

// ServiceInspectRaw 获取服务原始配置。
func (s *SwarmService) ServiceInspectRaw(ctx context.Context, id string) (dockerSwarm.Service, error) {
	svc, _, err := s.client.ServiceInspectWithRaw(ctx, id, dockerSwarm.ServiceInspectOptions{InsertDefaults: true})
	if err != nil {
		logman.Error("InspectService failed", "id", id, "error", err)
		return dockerSwarm.Service{}, err
	}
	return svc, nil
}

// ─── 辅助函数 ───

func (s *SwarmService) serviceCreateOptions(spec dockerSwarm.ServiceSpec) dockerSwarm.ServiceCreateOptions {
	return dockerSwarm.ServiceCreateOptions{
		EncodedRegistryAuth: s.serviceRegistryAuth(spec),
		QueryRegistry:       true,
	}
}

func (s *SwarmService) serviceUpdateOptions(spec dockerSwarm.ServiceSpec) dockerSwarm.ServiceUpdateOptions {
	return dockerSwarm.ServiceUpdateOptions{
		EncodedRegistryAuth: s.serviceRegistryAuth(spec),
		RegistryAuthFrom:    dockerSwarm.RegistryAuthFromPreviousSpec,
		QueryRegistry:       true,
	}
}

func (s *SwarmService) serviceRegistryAuth(spec dockerSwarm.ServiceSpec) string {
	if s.registryAuth == nil {
		return ""
	}
	imageRef := serviceImageRef(spec)
	if imageRef == "" {
		return ""
	}
	return s.registryAuth(imageRef)
}

func serviceImageRef(spec dockerSwarm.ServiceSpec) string {
	if spec.TaskTemplate.ContainerSpec != nil {
		return spec.TaskTemplate.ContainerSpec.Image
	}
	if spec.TaskTemplate.PluginSpec != nil {
		return spec.TaskTemplate.PluginSpec.Remote
	}
	return ""
}
