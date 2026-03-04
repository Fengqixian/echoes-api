package handler

import (
	"echoes-api/cmd/ws-client/script"
	v1 "echoes-api/echoes-api/v1"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// CommandMessage 定义服务端->客户端的统一消息体
// 例如: {"id":"123","cmd":"notify","payload":{"text":"hello"}}
type CommandMessage = v1.CommandMessage

// ResponseMessage 定义客户端->服务端的响应消息体
// 例如: {"id":"123","cmd":"notify","ok":true,"data":{...}}
type ResponseMessage = v1.ResponseMessage

type notifyPayload struct {
	Text string `json:"text"`
}

type echoPayload struct {
	Text string `json:"text"`
}

// dispatcher 将不同指令路由到对应处理函数
type handlerFunc func(*websocket.Conn, v1.CommandMessage) error

func BuildHandlers() map[string]handlerFunc {
	return map[string]handlerFunc{
		v1.Commands.Notify:               HandleNotify,
		v1.Commands.Echo:                 HandleEcho,
		v1.Commands.Ping:                 HandlePing,
		v1.Commands.Time:                 HandleTime,
		v1.Commands.Close:                HandleClose,
		v1.Commands.Welcome:              HandleWelcome,
		v1.Commands.LoginCheck:           HandleLoginCheck,
		v1.Commands.Release:              HandleRelease,
		v1.Commands.LoadAICoinNews:       HandleLoadAICoinNews,
		v1.Commands.LoadAICoinNewsDetail: HandleLoadAICoinNewsDetail,
	}
}

func WriteJSON(conn *websocket.Conn, v interface{}) error {
	_ = conn.SetWriteDeadline(time.Now().Add(v1.WriteWait))
	return conn.WriteJSON(v)
}

func HandleNotify(conn *websocket.Conn, msg v1.CommandMessage) error {
	var p notifyPayload
	if len(msg.Payload) > 0 {
		if err := json.Unmarshal([]byte(msg.Payload), &p); err != nil {
			return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: false, Error: "invalid payload"})
		}
	}
	fmt.Printf("[notify] %s\n", p.Text)
	return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: true})
}

func HandleEcho(conn *websocket.Conn, msg v1.CommandMessage) error {
	var p echoPayload
	if len(msg.Payload) > 0 {
		if err := json.Unmarshal([]byte(msg.Payload), &p); err != nil {
			return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: false, Error: "invalid payload"})
		}
	}
	return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: true, Data: map[string]string{"text": p.Text}})
}

func HandlePing(conn *websocket.Conn, msg v1.CommandMessage) error {
	// 接到服务端 ping 指令时，立即回发 pong 响应消息体（非 WebSocket 控制帧）
	return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: true, Data: "pong"})
}

func HandleTime(conn *websocket.Conn, msg v1.CommandMessage) error {
	now := time.Now().Format(time.RFC3339)
	return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: true, Data: map[string]string{"now": now}})
}

func HandleClose(conn *websocket.Conn, msg v1.CommandMessage) error {
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
	return nil
}

func HandleWelcome(conn *websocket.Conn, msg v1.CommandMessage) error {
	return nil
}

func HandleLoginCheck(conn *websocket.Conn, msg v1.CommandMessage) error {
	loggedIn, err := script.CheckLogin()
	if err != nil {
		log.Printf("CheckLogin failed: %s", err.Error())
		return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: false, Error: err.Error()})
	}

	return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: true, Data: map[string]bool{"loggedIn": loggedIn}})
}
func HandleRelease(conn *websocket.Conn, msg v1.CommandMessage) error {
	log.Printf("[release] %s\n", msg.Payload)
	var p v1.ReleaseParams
	if len(msg.Payload) > 0 {
		if err := json.Unmarshal([]byte(msg.Payload), &p); err != nil {
			return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: false, Error: "invalid payload"})
		}
	}

	if err := script.Release(p); err != nil {
		log.Printf("Release failed: %s", err.Error())
		return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: false, Error: err.Error()})
	}

	return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: true, Data: "发布成功"})
}

func HandleLoadAICoinNews(conn *websocket.Conn, msg v1.CommandMessage) error {
	log.Printf("[loadAICoinNews] %s\n", msg.Payload)
	news, err := script.LoadAICoinNews()
	if err != nil {
		log.Printf("LoadAICoinNews failed: %s", err.Error())
		return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: false, Error: err.Error()})
	}
	return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: true, Data: news})
}

func HandleLoadAICoinNewsDetail(conn *websocket.Conn, msg v1.CommandMessage) error {
	log.Printf("[loadAICoinNewsDetail] %s\n", msg.Payload)
	news, err := script.LoadAICoinNewsDetail(msg.Payload)
	if err != nil {
		log.Printf("LoadAICoinNewsDetail failed: %s", err.Error())
		return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: false, Error: err.Error()})
	}

	return WriteJSON(conn, ResponseMessage{ID: msg.ID, Cmd: msg.Cmd, OK: true, Data: news})
}
