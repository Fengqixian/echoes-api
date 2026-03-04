package script

import (
	"context"
	v1 "echoes-api/echoes-api/v1"
	ichromedp2 "echoes-api/ichromedp"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

var (
	chromeRemotePageInfoUrl = "http://127.0.0.1:9222/json"
	homeURL                 string
	once                    sync.Once
)

func SetHomeURL(url string) {
	once.Do(func() {
		homeURL = url
		log.Println("首页地址已设置:", homeURL)
	})
}

func GetHomeURL() string {
	return homeURL
}

type ChromeRemotePageInfo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Url         string `json:"url"`
}

func getChromeRemotePage(url string) (ChromeRemotePageInfo, error) {
	// 1. 创建一个具有超时设置的 HTTP 客户端
	client := http.Client{
		Timeout: 5 * time.Second, // 设置 5 秒超时
	}

	// 2. 发送 GET 请求
	resp, err := client.Get(url)
	if err != nil {
		return ChromeRemotePageInfo{}, fmt.Errorf("发送请求失败: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("关闭响应体失败: %v", err)
		}
	}(resp.Body) // 确保在函数结束时关闭响应体

	// 3. 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return ChromeRemotePageInfo{}, fmt.Errorf("请求返回非 200 状态码: %d", resp.StatusCode)
	}

	// 4. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChromeRemotePageInfo{}, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 5. 解析 JSON 响应
	var pages []ChromeRemotePageInfo
	if err := json.Unmarshal(body, &pages); err != nil {
		return ChromeRemotePageInfo{}, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	// 6. 检查是否有可用的页面信息
	if len(pages) == 0 {
		return ChromeRemotePageInfo{}, fmt.Errorf("JSON 响应中没有找到任何可用的 CDP 页面信息")
	}

	// 7. 返回第一个页面的 ID
	for _, page := range pages {
		if page.Title != "DevTools" {
			return page, nil
		}
	}

	return ChromeRemotePageInfo{}, fmt.Errorf("JSON 响应中没有找到任何可用的 CDP 页面信息")
}

func GetChromePath() string {
	devChrome := filepath.Join(v1.ExeDir(), "chrome", "chrome.exe")
	if v1.IsExist(devChrome) {
		log.Printf("Chrome 浏览器: %s", devChrome)
		return devChrome
	}

	defChrome := "C:/Program Files/Google/Chrome/Application/chrome.exe"
	if v1.IsExist(defChrome) {
		log.Printf("Chrome 浏览器: %s", defChrome)
		return defChrome
	}

	log.Printf("未找到 Chrome 可执行文件: %s %s", defChrome, defChrome)
	panic(fmt.Sprintf("未找到 Chrome 浏览器"))
}

func GetNewChromedpContext() (context.Context, context.CancelFunc) {
	ctx, cancel, err := ichromedp2.New(ichromedp2.NewConfig(
		ichromedp2.WithChromeFlags(chromedp.WindowSize(1920, 1080)),
		ichromedp2.WithChromeBinary(GetChromePath()),
		ichromedp2.WithTimeout(1*time.Minute),
		ichromedp2.WithHeadless(),
	))
	if err != nil {
		panic(err)
	}
	cancelFunc := func() {
		cancel()
	}

	return ctx, cancelFunc
}

func GetNewRemoteChromedpContext() (context.Context, context.CancelFunc) {
	page, err := getChromeRemotePage(chromeRemotePageInfoUrl)
	if err != nil {
		log.Printf("获取 Chrome 远程页面失败: %v", err)
		return nil, nil
	}

	SetHomeURL(page.Url)
	ctx, cancel, err := ichromedp2.New(ichromedp2.NewConfig(
		ichromedp2.WithChromeFlags(chromedp.WindowSize(1920, 1080)),
		ichromedp2.WithChromeRemote(chromeRemotePageInfoUrl),
		ichromedp2.WithTargetID(page.ID),
		ichromedp2.WithTimeout(1*time.Minute),
		ichromedp2.WithHeadless(),
	))
	if err != nil {
		log.Printf("创建 Chromedp 上下文失败: %v", err)
		return nil, nil
	}
	cancelFunc := func() {
		cancel()
	}

	return ctx, cancelFunc
}

func BackRpaClientHome() {
	if GetHomeURL() == "" {
		return
	}

	ctx, _ := GetNewRemoteChromedpContext()
	err := chromedp.Run(ctx, chromedp.Navigate(GetHomeURL()))
	if err != nil {
		log.Printf("返回 RPA 客户端首页失败: %v", err)
		return
	}
	log.Printf("返回 RPA 客户端首页成功: %s", GetHomeURL())
}
