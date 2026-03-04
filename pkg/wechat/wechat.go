package wechat

import (
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/spf13/viper"
)

type Wechat struct {
	Ws  *wechat.Wechat
	Cfg *offConfig.Config
}

func NewWechat(conf *viper.Viper) *Wechat {
	wc := wechat.NewWechat()
	memory := cache.NewMemory()
	cfg := &offConfig.Config{
		AppID:          conf.GetString("wechat.app_id"),
		AppSecret:      conf.GetString("wechat.app_secret"),
		Token:          conf.GetString("wechat.token"),
		EncodingAESKey: conf.GetString("wechat.encoding_aes_key"),
		Cache:          memory,
	}

	return &Wechat{Ws: wc, Cfg: cfg}
}

func (w *Wechat) GetAccessToken() (string, error) {
	officialAccount := w.Ws.GetOfficialAccount(w.Cfg)
	return officialAccount.GetAccessToken()
}
