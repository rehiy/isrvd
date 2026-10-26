# GPU 监控

支持自动检测 NVIDIA / AMD / Intel / Apple Silicon GPU，显示使用率、显存、温度、功耗、风扇转速。

| 厂商 | 首选方式 | 回退方式 |
| ------ | --------- | --------- |
| NVIDIA | nvidia-smi | — |
| AMD | sysfs | rocm-smi |
| Intel | sysfs（依赖内核驱动是否暴露相应节点，缺失时为 0） | — |
| Apple Silicon | ioreg（统一内存，显存值为系统内存占用） | — |

虚拟显卡按 PCI 厂商 ID 过滤；Intel 核显按产品名（HD/UHD/Iris/GMA）过滤，名称含 Arc 的保留；无独立显存的 AMD APU 通常会在采集时自然跳过。

> 温度、功耗、风扇转速并非所有平台都能取到，不可用时为 `-1`：Apple Silicon 的温度、功耗与风扇恒不可用；Intel 恒无风扇转速；AMD 走 rocm-smi 回退时同样无风扇转速。利用率、显存读取失败时为 `0`。

## 进程级占用

进程列表（`GET /local/processes`）会附带每个进程的 GPU 显存、利用率与所占用设备，字段见 [本机进程管理 API](references/local.md)。按进程逐个采集，结果缓存 2 秒：

- Linux 首选内核 DRM fdinfo：扫描 `/proc/<pid>/fd{,info}` 指向的 `/dev/dri/*`，解析 `drm-memory-*`、`drm-engine-*`，与显卡厂商无关；利用率由相邻两次采样差分得出，首次不返回
- 闭源驱动未暴露 fdinfo 的进程，由 `nvidia-smi --query-compute-apps=pid,used_memory,gpu_uuid` 补齐：仅覆盖 compute 进程，不提供利用率
- 非 Linux 仅有 nvidia-smi；Linux 需要能访问 `/dev/dri`，缺失时跳过 DRM 扫描，仅剩 nvidia-smi 兜底

## 容器部署注意事项

**NVIDIA**：

```bash
docker run -d --gpus all rehiy/isrvd:slim
```

**AMD / Intel**：

```bash
docker run -d --device /dev/dri:/dev/dri rehiy/isrvd:slim
```

> 进程级占用依赖 `/dev/dri`，AMD / Intel 需要上面的设备透传才能采集；NVIDIA 闭源驱动场景下由 `nvidia-smi` 提供进程级数据。
