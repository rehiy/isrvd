package app

import (
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"

	"isrvd/server/config"
)

// 上传入口与表单解析放在同一文件，供 OpenAPI 生成器跟进 multipart 字段。
func (app *App) dockerContainerFileUpload(c *gin.Context) {
	id := c.Param("id")
	uploadFileBatch(c, http.StatusInternalServerError, func(dirPath, fileName string, file multipart.File) error {
		return app.dockerSvc.ContainerFileUpload(c.Request.Context(), id, dirPath, fileName, file)
	})
}

func (app *App) websshSFTPUpload(c *gin.Context) {
	id := c.Param("id")
	uploadFileBatch(c, http.StatusBadRequest, func(dirPath, fileName string, file multipart.File) error {
		return app.websshSvc.SFTPUpload(id, dirPath, file, fileName)
	})
}

// ─── 辅助函数 ───

func uploadFileBatch(c *gin.Context, uploadErrorStatus int, upload func(dirPath, fileName string, file multipart.File) error) {
	dirPath := c.Query("path")
	if dirPath == "" {
		respondError(c, http.StatusBadRequest, "path 参数不能为空")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, config.Current().Server.MaxUploadSize)
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		respondError(c, http.StatusBadRequest, "解析上传表单失败: "+err.Error())
		return
	}
	files := c.Request.MultipartForm.File["file"]
	if len(files) == 0 {
		respondError(c, http.StatusBadRequest, "未找到上传文件")
		return
	}

	// 目录上传的 relativePath 与 file 一一对应；缺失时沿用文件名。
	relativePaths := c.Request.MultipartForm.Value["relativePath"]
	for i, header := range files {
		fileName := header.Filename
		if i < len(relativePaths) && relativePaths[i] != "" {
			fileName = relativePaths[i]
		}
		file, err := header.Open()
		if err != nil {
			respondError(c, http.StatusBadRequest, "打开文件失败: "+err.Error())
			return
		}
		err = upload(dirPath, fileName, file)
		file.Close()
		if err != nil {
			respondError(c, uploadErrorStatus, err.Error())
			return
		}
	}
	respondSuccess(c, "上传成功", nil)
}
