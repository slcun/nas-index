package terminal

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HandleWebSocket 处理 WebSocket 连接，创建真实 PTY 终端
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade 失败: %v", err)
		return
	}
	defer conn.Close()

	cols, rows := parseSizeFromQuery(r)

	term, err := NewTerminal(cols, rows)
	if err != nil {
		log.Printf("创建终端失败: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("\r\n创建终端失败: "+err.Error()+"\r\n"))
		return
	}
	defer term.Close()

	// PTY -> WebSocket：读取终端输出并发送给客户端
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := term.Read(buf)
			if err != nil {
				select {
				case <-done:
				default:
					log.Printf("PTY 读取结束: %v", err)
					conn.WriteMessage(websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseNormalClosure, "PTY 已关闭"))
				}
				return
			}
			if n > 0 {
				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
		}
	}()

	// WebSocket -> PTY：接收客户端输入并写入终端
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			close(done)
			return
		}

		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			msg := string(p)

			// 处理调整大小的特殊消息
			if strings.HasPrefix(msg, "\x1b[RESIZE:") {
				handleResize(msg, term)
				continue
			}

			// 将用户输入写入 PTY
			if _, err := term.Write(p); err != nil {
				log.Printf("PTY 写入失败: %v", err)
				return
			}
		}
	}
}

// parseSizeFromQuery 从查询参数解析终端大小
func parseSizeFromQuery(r *http.Request) (uint16, uint16) {
	cols := uint16(80)
	rows := uint16(24)

	if c := r.URL.Query().Get("cols"); c != "" {
		if v, err := strconv.Atoi(c); err == nil && v > 0 {
			cols = uint16(v)
		}
	}
	if ro := r.URL.Query().Get("rows"); ro != "" {
		if v, err := strconv.Atoi(ro); err == nil && v > 0 {
			rows = uint16(v)
		}
	}

	return cols, rows
}

// handleResize 解析 RESIZE 消息并调整终端大小
// 消息格式: \x1b[RESIZE:cols;rows]
func handleResize(msg string, term *Terminal) {
	msg = strings.TrimPrefix(msg, "\x1b[RESIZE:")
	msg = strings.TrimSuffix(msg, "]")
	parts := strings.Split(msg, ";")
	if len(parts) != 2 {
		return
	}
	cols, err1 := strconv.Atoi(parts[0])
	rows, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return
	}
	if cols > 0 && rows > 0 {
		term.Resize(cols, rows)
	}
}
