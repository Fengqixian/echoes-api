package script

import (
	v1 "echoes-api/echoes-api/v1"
	"errors"
	"log"

	"github.com/chromedp/chromedp"
)

var baseUrl = "https://www.aicoin.com/zh-Hans"

var aicoinNewsPage = baseUrl + "/news/all"

// 目标 class（注意 class 含有多个，用空格分隔，chromedp 推荐用 CSS 选择器）
var selector = `a.group.relative.col-span-1.col-start-2.row-span-1.row-start-2.w-\[130px\].cursor-pointer.overflow-hidden.sm\:h-\[140px\].sm\:w-\[220px\]`

func LoadAICoinNewsDetail(url string) (v1.AICoinNews, error) {
	ctx, cancel := GetNewChromedpContext()
	defer cancel()

	item := v1.AICoinNews{Url: url}
	err := chromedp.Run(ctx, chromedp.Navigate(url),
		chromedp.WaitVisible(`h1.title`, chromedp.ByQuery),
		chromedp.Text(`header.article-header h1.title`, &item.Title, chromedp.NodeVisible, chromedp.ByQuery),
		chromedp.WaitVisible(`div.article-content.content`, chromedp.ByQuery),
		chromedp.InnerHTML(`div.article-content.content`, &item.Content, chromedp.ByQuery),
	)
	if err != nil {
		log.Printf("[loadAICoinNewsDetail] %s\n", err)
		return item, errors.New("load aicoin news detail failed")
	}

	return item, nil
}

func LoadAICoinNews() ([]string, error) {
	ctx, cancel := GetNewChromedpContext()
	defer cancel()

	var hrefs []map[string]string
	err := chromedp.Run(ctx, chromedp.Navigate(aicoinNewsPage),
		// 等待页面主要内容加载完成（可根据实际情况调整）
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		// 获取所有匹配的 a 标签的 href
		chromedp.AttributesAll(selector, &hrefs, chromedp.ByQueryAll))
	if err != nil {
		log.Printf("[loadAICoinNews] %s\n", err)
		return nil, errors.New("load aicoin news failed")
	}

	news := make([]string, 0)

	// 打印所有 href
	for _, href := range hrefs {
		if href["href"] == "" {
			continue
		}

		news = append(news, baseUrl+href["href"])
	}

	return news, nil
}
