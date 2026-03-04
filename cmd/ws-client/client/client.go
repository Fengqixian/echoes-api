package client

import (
	"echoes-api/cmd/ws-client/handler"
	v1 "echoes-api/echoes-api/v1"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxReconnectInterval     = 60 * time.Second // 最大重连间隔
	initialReconnectInterval = 2 * time.Second  // 初始重连间隔
)

// Client WebSocket 客户端
type Client struct {
	addr              string
	token             string
	interrupt         chan os.Signal
	shouldReconnect   bool
	shouldReconnectMu sync.Mutex
	reconnectInterval time.Duration
}

// NewClient 创建新的 WebSocket 客户端
func NewClient(addr, token string) *Client {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	return &Client{
		addr:              addr,
		token:             token,
		interrupt:         interrupt,
		shouldReconnect:   true,
		reconnectInterval: initialReconnectInterval,
	}
}

// Connect 建立 WebSocket 连接
func (c *Client) connect() (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	headers := http.Header{}
	headers.Set("Authorization", c.token)

	conn, resp, err := dialer.Dial(c.addr, headers)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("dial error: %s (status=%d)", string(body), resp.StatusCode)
		}
		return nil, fmt.Errorf("dial error: %s", err)
	}

	// 心跳：读超时 + pong 续期
	_ = conn.SetReadDeadline(time.Now().Add(v1.PongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(v1.PongWait))
	})

	return conn, nil
}

// handleConnection 处理 WebSocket 连接，返回 true 表示需要重连，false 表示正常退出
func (c *Client) handleConnection(conn *websocket.Conn) bool {
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Printf("关闭连接错误: %v", err)
		}
	}()

	// 用于协调 goroutine 退出
	done := make(chan struct{})
	var wg sync.WaitGroup

	// 定时发送 Ping
	ticker := time.NewTicker(v1.PingPeriod)
	wg.Add(1)
	go func() {
		defer ticker.Stop()
		defer wg.Done()
		for {
			select {
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(v1.WriteWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("ping error: %v", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// 读循环：解析文本消息为指令并分发
	wg.Add(1)
	go func() {
		defer wg.Done()
		handlers := handler.BuildHandlers()
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				// 检查是否是正常关闭
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("read error: %v", err)
				} else {
					log.Printf("连接关闭: %v", err)
				}
				close(done)
				return
			}
			switch msgType {
			case websocket.TextMessage:
				log.Printf("received message: %s", string(data))
				var cmd v1.CommandMessage
				if err := json.Unmarshal(data, &cmd); err != nil {
					log.Printf("invalid message json: %s", string(data))
					_ = handler.WriteJSON(conn, handler.ResponseMessage{OK: false, Error: "invalid json"})
					continue
				}

				h, ok := handlers[cmd.Cmd]
				if !ok {
					log.Printf("unknown command: %s", cmd.Cmd)
					continue
				}

				if err := h(conn, cmd); err != nil {
					log.Printf("handler error for cmd=%s: %v", cmd.Cmd, err)
				}

				// script.BackRpaClientHome()
			default:
				fmt.Printf("<- [type=%d] %d bytes\n", msgType, len(data))
			}
		}
	}()

	// 等待退出信号或连接断开
	select {
	case <-done:
		log.Println("连接已断开")
		// 等待所有 goroutine 完成
		wg.Wait()
		// 检查是否应该重连
		c.shouldReconnectMu.Lock()
		reconnect := c.shouldReconnect
		c.shouldReconnectMu.Unlock()
		return reconnect
	case <-c.interrupt:
		log.Println("收到中断信号，正在关闭连接...")
		c.shouldReconnectMu.Lock()
		c.shouldReconnect = false
		c.shouldReconnectMu.Unlock()

		// 优雅关闭
		_ = conn.SetWriteDeadline(time.Now().Add(v1.WriteWait))
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		time.Sleep(500 * time.Millisecond)

		close(done)
		wg.Wait()
		return false
	}
}

// Run 启动客户端连接循环（支持断线重连）
func (c *Client) Run() {
	for {
		// 检查是否应该继续重连
		c.shouldReconnectMu.Lock()
		reconnect := c.shouldReconnect
		c.shouldReconnectMu.Unlock()

		if !reconnect {
			log.Println("已停止重连，程序退出")
			return
		}

		// 建立连接
		conn, err := c.connect()
		if err != nil {
			log.Printf("连接失败: %v", err)
			log.Printf("将在 %v 后重试...", c.reconnectInterval)
			select {
			case <-time.After(c.reconnectInterval):
				// 指数退避：每次重连失败后间隔加倍，但不超过最大值
				c.reconnectInterval *= 2
				if c.reconnectInterval > maxReconnectInterval {
					c.reconnectInterval = maxReconnectInterval
				}
				continue
			case <-c.interrupt:
				log.Println("收到中断信号，停止重连")
				return
			}
		}

		log.Println("WebSocket 连接成功")
		// 连接成功后重置重连间隔
		c.reconnectInterval = initialReconnectInterval

		// 处理连接，如果连接断开则返回 false 表示需要重连
		needsReconnect := c.handleConnection(conn)

		if !needsReconnect {
			log.Println("连接已关闭，退出程序")
			return
		}

		// 连接断开后，等待一段时间再重连（使用当前的重连间隔）
		log.Printf("连接已断开，将在 %v 后尝试重连...", c.reconnectInterval)
		select {
		case <-time.After(c.reconnectInterval):
			// 指数退避：每次重连失败后间隔加倍，但不超过最大值
			c.reconnectInterval *= 2
			if c.reconnectInterval > maxReconnectInterval {
				c.reconnectInterval = maxReconnectInterval
			}
		case <-c.interrupt:
			log.Println("收到中断信号，停止重连")
			c.shouldReconnectMu.Lock()
			c.shouldReconnect = false
			c.shouldReconnectMu.Unlock()
			return
		}
	}
}
