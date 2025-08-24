package handler

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Shell处理器
type ShellHandler struct{}

// 创建Shell处理器
func NewShellHandler() *ShellHandler {
	return &ShellHandler{}
}

// Shell WebSocket处理
func (h *ShellHandler) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	// 启动 shell 进程，优先 bash，失败降级 sh
	// 使用 pty 启动 shell，支持交互
	cmd := exec.Command("bash")
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	cmd.Env = append(cmd.Env, os.Environ()...)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Println("start bash error, try sh:", err)
		cmd = exec.Command("sh")
		cmd.Env = append(cmd.Env, os.Environ()...)
		ptmx, err = pty.Start(cmd)
		if err != nil {
			log.Println("start shell error:", err)
			return
		}
	}
	defer cmd.Process.Kill()
	defer ptmx.Close()

	// 读取 shell 输出并发给前端
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				conn.WriteMessage(websocket.TextMessage, buf[:n])
			}
			if err != nil {
				if err != io.EOF {
					log.Println("pty read error:", err)
				}
				break
			}
		}
	}()

	// 连接成功后，主动推送一条欢迎信息
	conn.WriteMessage(websocket.TextMessage, []byte("[终端已连接，输入命令后回车]\r\n"))

	// 读取前端输入并写入 shell
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			break
		}
		_, err = ptmx.Write(msg)
		if err != nil {
			log.Println("pty write error:", err)
			break
		}
	}
}
