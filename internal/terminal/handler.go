package terminal

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该限制
	},
}

// HandleWebSocket 处理 WebSocket 连接
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// 发送欢迎消息
	welcomeMsg := "\r\nWelcome to NAS Manager Terminal!\r\n\r\nThis is a demo terminal. Full PTY support coming soon.\r\n\r\n$ "
	conn.WriteMessage(websocket.TextMessage, []byte(welcomeMsg))

	// 简单的模拟终端
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}

		if messageType == websocket.TextMessage {
			msg := string(p)
			
			// 处理调整大小的消息
			if strings.HasPrefix(msg, "\x1b[RESIZE:") {
				continue
			}

			// 回显输入
			conn.WriteMessage(websocket.TextMessage, p)
			
			// 处理回车键
			if strings.Contains(msg, "\r") || strings.Contains(msg, "\n") {
				// 简单的命令响应
				response := "\r\nCommand executed (demo mode)\r\n$ "
				conn.WriteMessage(websocket.TextMessage, []byte(response))
			}
		}
	}
}
