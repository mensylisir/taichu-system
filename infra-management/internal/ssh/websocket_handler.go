package ssh

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// WebSocketSSHHandler WebSocket SSH处理器
type WebSocketSSHHandler struct {
	upgrader websocket.Upgrader
}

// NewWebSocketSSHHandler 创建WebSocket SSH处理器
func NewWebSocketSSHHandler() *WebSocketSSHHandler {
	return &WebSocketSSHHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 开发环境，生产环境需要限制
			},
		},
	}
}

// HandleWebSocket 处理WebSocket连接
func (h *WebSocketSSHHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级为WebSocket连接
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	// 获取连接参数
	query := r.URL.Query()
	host := query.Get("host")
	port := query.Get("port")
	username := query.Get("username")
	password := query.Get("password")

	if host == "" || username == "" || password == "" {
		err := conn.WriteMessage(websocket.TextMessage, []byte("错误: 缺少必要参数"))
		if err != nil {
			log.Printf("发送错误消息失败: %v", err)
		}
		return
	}

	portInt := 22 // 默认SSH端口
	if port != "" {
		fmt.Sscanf(port, "%d", &portInt)
	}

	// 创建SSH连接
	manager := NewSSHManager(username, password)
	client, err := manager.Connect(host, portInt)
	if err != nil {
		errMsg := fmt.Sprintf("SSH连接失败: %v", err)
		err := conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
		if err != nil {
			log.Printf("发送错误消息失败: %v", err)
		}
		return
	}
	defer manager.Close(client)

	// 创建SSH会话
	sshConn, err := NewSSHConnection(client)
	if err != nil {
		errMsg := fmt.Sprintf("创建SSH会话失败: %v", err)
		err := conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
		if err != nil {
			log.Printf("发送错误消息失败: %v", err)
		}
		return
	}
	defer sshConn.Close()

	// 开始SSH会话
	if err := sshConn.session.Shell(); err != nil {
		errMsg := fmt.Sprintf("启动SSH Shell失败: %v", err)
		err := conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
		if err != nil {
			log.Printf("发送错误消息失败: %v", err)
		}
		return
	}

	// 发送连接成功消息
	successMsg := fmt.Sprintf("已连接到 %s@%s", username, host)
	err = conn.WriteMessage(websocket.TextMessage, []byte(successMsg))
	if err != nil {
		log.Printf("发送成功消息失败: %v", err)
		return
	}

	// 启动goroutine转发SSH输出到WebSocket
	go func() {
		defer conn.Close()
		defer sshConn.Close()

		// 读取SSH标准输出
		for {
			buf := make([]byte, 1024)
			n, err := sshConn.GetStdout().Read(buf)
			if err != nil {
				log.Printf("SSH stdout读取失败: %v", err)
				return
			}
			if n > 0 {
				data := string(buf[:n])
				err := conn.WriteMessage(websocket.TextMessage, []byte(data))
				if err != nil {
					log.Printf("WebSocket发送SSH输出失败: %v", err)
					return
				}
			}
		}
	}()

	// 启动goroutine转发SSH错误输出到WebSocket
	go func() {
		defer conn.Close()
		defer sshConn.Close()

		// 读取SSH标准错误输出
		for {
			buf := make([]byte, 1024)
			n, err := sshConn.GetStderr().Read(buf)
			if err != nil {
				log.Printf("SSH stderr读取失败: %v", err)
				return
			}
			if n > 0 {
				data := string(buf[:n])
				err := conn.WriteMessage(websocket.TextMessage, []byte(data))
				if err != nil {
					log.Printf("WebSocket发送SSH错误输出失败: %v", err)
					return
				}
			}
		}
	}()

	// WebSocket -> SSH
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket读取消息失败: %v", err)
			break
		}

		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			// 处理xterm.js的输入数据
			if len(message) > 0 {
				// 写入SSH
				_, err := sshConn.GetStdin().Write(message)
				if err != nil {
					log.Printf("写入SSH失败: %v", err)
					break
				}
			}
		}
	}
}