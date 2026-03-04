package main

import (
	"echoes-api/cmd/ws-client/script"
	"log"
	"os"

	"github.com/chromedp/chromedp"
)

// Build cmd: go build -ldflags="-s -w" -o .\cmd\ws-client\xhs-rpa.exe .\cmd\ws-client\main.go
/*func main() {
	addr := flag.String("url", "wss://www.hookfunc.com/rpa/v1/ws", "websocket url")
	token := flag.String("token", "bbf6ccfe-b1f7-4857-ba9e-b362ef10d010", "Authorization header value")
	flag.Parse()

	if strings.TrimSpace(*token) == "" {
		log.Fatalf("Authorization token is required. Use -token=<your-token>")
	}

	// 创建客户端并运行
	wsClient := client.NewClient(*addr, *token)
	wsClient.Run()
}*/

func main() {
	ctx, cancel := script.GetNewChromedpContext()
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx, chromedp.Navigate("http://localhost:5173/"),
		chromedp.WaitVisible("body", chromedp.ByQuery),   // 等待页面基本加载
		chromedp.WaitVisible("#chart", chromedp.ByQuery), // 等待目标元素可见
		chromedp.Screenshot("#chart", &buf, chromedp.NodeVisible),
	)
	if err != nil {
		return
	}

	if err := os.WriteFile("element_screenshot.png", buf, 0o644); err != nil {
		log.Fatal(err)
	}

}
