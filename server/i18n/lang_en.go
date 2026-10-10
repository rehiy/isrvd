package i18n

// 英文：以中文原文为 key 的译文。
//
// 中文是代码里书写提示文案的语言，也是所有语言词典共同的 key；
// 未登记的文案回落中文原文，不会丢失信息。动词只使用 %s / %d / %q / %v / %w / %T，
// 且译文的动词序列必须与 key 一致（由 TestCatalogs 自检）。
//
// 新增语言：参照本文件新建 lang_<code>.go，声明 Language 并登记译文即可，无需改动其他代码。

// langEN 英文
var langEN = Language{Code: "en", Tags: []string{"en"}}

func init() {
	Register(langEN, map[string]string{
		// ─── 通用结果 ───
		"请求失败":     "request failed",
		"操作成功":     "operation succeeded",
		"操作失败":     "operation failed",
		"查询成功":     "query succeeded",
		"当前已是最新版本": "already up to date",
		"接口不存在":    "api not found",
		"请求体格式错误":  "invalid request",
		"请求参数错误":   "invalid parameters",
		"资源不存在":    "not found",

		// ─── 服务状态 ───
		"服务正在重载":       "service is reloading",
		"服务不可用":        "service unavailable",
		"配置正在重载，请稍后重试": "config is reloading, please retry later",
		"监控采集器未启动":     "monitor collector not started",

		// ─── 认证与权限 ───
		"请先登录":          "please login first",
		"登录成功":          "login succeeded",
		"登录已过期，请重新登录":   "login expired, please login again",
		"用户名或密码错误":      "invalid credentials",
		"认证令牌无效":        "invalid token",
		"验证码无效":         "invalid verification code",
		"权限不足":          "insufficient permissions",
		"未授权的访问路径":      "unauthorized path",
		"路径越界，拒绝访问":     "path is out of bounds, denied",
		"用户不存在":         "user not found",
		"用户名不能为空":       "username is required",
		"密码不能为空":        "password is required",
		"密码长度不能少于 %d 位": "password must be at least %d chars",
		"密码加密失败":        "password encryption failed",
		"密码修改成功":        "password changed",
		"验证失败":          "verification failed",

		// ─── 参数校验 ───
		"path 参数不能为空":  "path parameter is required",
		"index 必须为整数":  "index must be an integer",
		"ID 不能为空":      "id is required",
		"缺少容器 ID":      "container id is required",
		"容器ID不能为空":     "container id cannot be empty",
		"镜像ID不能为空":     "image id cannot be empty",
		"镜像名称不能为空":     "image name is required",
		"网络ID不能为空":     "network id cannot be empty",
		"缺少服务 ID":      "service id is required",
		"路由 ID 不能为空":   "route id is required",
		"路由名称不能为空":     "route name is required",
		"缺少 url 参数":    "url parameter is required",
		"缺少 sessionId": "session id is required",
		"缺少 runId":     "run id is required",
		"进程 ID 无效":     "process id is invalid",
		"操作类型不能为空":     "operation type is required",
		"未知部署目标":       "unknown deploy target",
		"未配置容器数据根目录":   "container data root missing",

		// ─── 文件 ───
		"文件超过在线编辑上限，请使用上传功能": "file exceeds online editing limit, please upload",
		"文件超过在线编辑上限，请使用下载功能": "file exceeds online editing limit, please download",
		"文件大小超过限制":           "file size exceeds the limit",
		"文件未找到":              "file not found",
		"目录不存在":              "directory does not exist",
		"未找到上传文件":            "no uploaded file found",
		"不支持预览的文件类型":         "preview not supported for this file type",
		"无效的目录名，不允许包含路径分隔符":  "invalid directory name, separator not allowed",
		"无法创建目录":             "failed to create directory",
		"无法创建文件":             "failed to create file",
		"无法写入文件":             "failed to write file",
		"无法保存文件":             "failed to save file",
		"无法删除文件":             "failed to delete file",
		"无法重命名文件":            "failed to rename file",
		"无法解压文件":             "failed to extract file",
		"无法创建压缩文件":           "failed to create archive",
		"无法计算目录大小":           "failed to calculate directory size",
		"解析上传表单失败":           "failed to parse upload form",
		"文件保存成功":             "file saved",
		"文件创建成功":             "file created",
		"文件删除成功":             "file deleted",
		"文件重命名成功":            "file renamed",
		"文件解压成功":             "file extracted",
		"目录创建成功":             "directory created",
		"权限修改成功":             "permissions changed",
		"所有者修改成功":            "owner changed",

		// ─── 资源操作结果 ───
		"创建成功":     "created successfully",
		"更新成功":     "updated successfully",
		"删除成功":     "deleted successfully",
		"保存成功":     "saved successfully",
		"重命名成功":    "renamed successfully",
		"部署成功":     "deployed successfully",
		"重建成功":     "rebuilt successfully",
		"配置更新成功":   "configuration updated",
		"成员添加成功":   "member added",
		"成员更新成功":   "member updated",
		"成员删除成功":   "member deleted",
		"账号添加成功":   "account added",
		"账号删除成功":   "account deleted",
		"镜像仓库添加成功": "registry added",
		"镜像仓库更新成功": "registry updated",
		"镜像仓库删除成功": "registry deleted",
		"已发送终止信号":  "termination signal sent",
		"开始绑定":     "starting binding",
		"开始登录":     "starting login",
		"开始注册":     "starting registration",

		// ─── 中控网关 ───
		"不允许的跨站来源":      "cross-site origin not allowed",
		"仅创始人可管理和操作节点":  "only the founder can manage nodes",
		"不接受浏览器发起的隧道连接": "tunnel connections from browsers are not accepted",
		"请求体过大":         "request body too large",
		"请求体无效":         "invalid request body",

		// ─── 资源查询 ───
		"证书不存在":             "certificate not found",
		"凭据不存在":             "credential not found",
		"主机不存在":             "host not found",
		"节点离线":              "node offline",
		"节点已审批":             "node approved",
		"节点已吊销":             "node revoked",
		"注册码已撤销":            "registration code revoked",
		"该路由未配置 Basic Auth": "basic auth not configured",
		"获取进程列表失败":          "failed to get process list",
		"获取进程列表成功":          "process list retrieved",
		"获取容器列表失败":          "failed to get container list",
	})
}
