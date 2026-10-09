# GPU 监控

支持自动检测 NVIDIA / AMD / Intel / Apple Silicon 独立显卡，显示使用率、显存、温度、功耗、风扇转速。

| 厂商 | 首选方式 | 回退方式 |
| ------ | --------- | --------- |
| NVIDIA | nvidia-smi | — |
| AMD | sysfs | rocm-smi |
| Intel | sysfs | — |
| Apple Silicon | ioreg | — |

自动过滤虚拟显卡和 Intel 核显（Intel Arc 独显保留）；无独立显存的 AMD APU 通常会在采集时自然跳过。

> 采集能力随厂商不同：温度、功耗、风扇转速并非所有平台都能取到。Apple Silicon 的温度、功耗与风扇恒为不可用（ioreg 只能取到利用率与显存）；Intel 恒无风扇转速；AMD 走 rocm-smi 回退时同样无风扇转速。不可用时返回 `-1`。

## 进程级占用

进程列表（`GET /local/processes`）会附带每个进程的 GPU 显存、利用率与所占用设备，字段见 [本机进程管理 API](references/local.md)。按进程逐个采集，结果缓存 2 秒：

- Linux 首选内核 DRM fdinfo：扫描 `/proc/<pid>/fd{,info}` 指向的 `/dev/dri/*`，解析 `drm-memory-*`、`drm-engine-*`，与显卡厂商无关
- 闭源驱动未暴露 fdinfo 的进程，由 `nvidia-smi --query-compute-apps=pid,used_memory,gpu_uuid` 补齐
- 需要能访问 `/dev/dri`；缺失时跳过 DRM 扫描，仅剩 nvidia-smi 兜底

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
