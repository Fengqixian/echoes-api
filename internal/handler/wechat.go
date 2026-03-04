package handler

import (
	"echoes-api/pkg/wechat"
	"github.com/gin-gonic/gin"
	"github.com/silenceper/wechat/v2/officialaccount/message"
	"go.uber.org/zap"
)

type WechatHandler struct {
	*Handler
	wechat *wechat.Wechat
}

func NewWechatHandler(
	handler *Handler,
	wechat *wechat.Wechat,
) *WechatHandler {
	return &WechatHandler{
		Handler: handler,
		wechat:  wechat,
	}
}

func (h *WechatHandler) MessageCallBack(ctx *gin.Context) {
	officialAccount := h.wechat.Ws.GetOfficialAccount(h.wechat.Cfg)
	server := officialAccount.GetServer(ctx.Request, ctx.Writer)
	server.SetMessageHandler(func(msg *message.MixMessage) *message.Reply {
		text := message.NewText(msg.Content)
		return &message.Reply{MsgType: message.MsgTypeText, MsgData: text}
	})

	err := server.Serve()
	if err != nil {
		h.logger.Error("serve message error", zap.Error(err))
		return
	}
	//发送回复的消息
	err = server.Send()
	if err != nil {
		h.logger.Error("send message error", zap.Error(err))
		return
	}
}
