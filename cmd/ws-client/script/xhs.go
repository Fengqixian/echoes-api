package script

import (
	"context"
	v1 "echoes-api/echoes-api/v1"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"
)

var (
	ReleasePage     = "https://creator.xiaohongshu.com/publish/publish?source=official&from=menu&target=image"
	ContentSelector = `div.tiptap.ProseMirror[contenteditable="true"][role="textbox"]`
	LoginPage       = "https://www.xiaohongshu.com/explore"
	CookiePath      = "cookie.txt"
)

func CheckLogin() (bool, error) {
	log.Println("检查登录是否过期")
	exist := v1.CheckCookieFileExist(CookiePath)
	if !exist {
		log.Println("cookie 文件不存在")
		return false, nil // 文件不存在
	}

	cookie, err := v1.ReadCookieFromFile(CookiePath)
	if err != nil {
		return false, err
	}

	if os.IsNotExist(err) {
		log.Println("cookie 文件内容为空")
		return false, nil // 文件不存在
	}

	ctx, cancel := GetNewChromedpContext()
	defer cancel()

	err = chromedp.Run(ctx,
		v1.SetCookie(cookie),
		chromedp.Navigate(LoginPage),
		// 加载首页
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("首页面已加载完成")
			return nil
		}),
		// 注入 div 元素
		chromedp.Evaluate(`
			var div = document.createElement('div');
			div.id = 'login-toast';
			div.innerText = '检查登录状态中！！！';
			document.body.appendChild(div);
		`, nil),
		// 注入 CSS 样式：红色字体、居中、浮动、醒目动画
		chromedp.Evaluate("var style = document.createElement('style');\n\t\t\tstyle.innerHTML = `\n\t\t#login-toast {\n\t\t\tposition: fixed;\n\t\t\ttop: 20%;\n\t\t\tleft: 50%;\n\t\t\ttransform: translate(-50%, -50%);\n\t\t\tz-index: 9999;\n\t\t\tfont-size: 32px;\n\t\t\tcolor: red;\n\t\t\tbackground-color: rgba(255, 255, 255, 0.8);\n\t\t\tpadding: 10px;\n\t\t\tborder-radius: 5px;\n\t\t\tanimation: pulse 1.5s infinite;\n\t\t}\n\t\t@keyframes pulse {\n\t\t\t0% { transform: translate(-50%, -50%) scale(1); opacity: 1; }\n\t\t\t50% { transform: translate(-50%, -50%) scale(1.2); opacity: 0.7; }\n\t\t\t100% { transform: translate(-50%, -50%) scale(1); opacity: 1; }\n\t\t}\n\t`;\n\t\t\tdocument.head.appendChild(style);", nil),
	)
	if err != nil {
		log.Println(fmt.Sprintf("%s: chromedp执行异常", LoginPage), zap.Error(err))
		return false, errors.New("登录检查失败")
	}

	log.Println("检查登录状态")
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 3*time.Second)
	defer timeoutCancel()
	var nodes []*cdp.Node
	_ = chromedp.Run(timeoutCtx, chromedp.Nodes(".main-container .user .link-wrapper .channel", &nodes, chromedp.ByQuery))
	if len(nodes) == 0 {
		log.Println("登录已过期")
		return false, nil
	}

	log.Println("已登录")
	return true, nil
}

func Login() error {
	log.Println("开始登录")
	timeoutCtx, cancel := GetNewChromedpContext()
	defer cancel()

	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(LoginPage),
		// 加载首页
		chromedp.WaitVisible("body", chromedp.ByQuery),
		// 注入 div 元素
		chromedp.Evaluate(`
			var div = document.createElement('div');
			div.id = 'login-toast';
			div.innerText = '请扫描二维码登录或输入手机号登录网页！！！';
			document.body.appendChild(div);
		`, nil),
		// 注入 CSS 样式：红色字体、居中、浮动、醒目动画
		chromedp.Evaluate("var style = document.createElement('style');\n\t\t\tstyle.innerHTML = `\n\t\t#login-toast {\n\t\t\tposition: fixed;\n\t\t\ttop: 20%;\n\t\t\tleft: 50%;\n\t\t\ttransform: translate(-50%, -50%);\n\t\t\tz-index: 9999;\n\t\t\tfont-size: 32px;\n\t\t\tcolor: red;\n\t\t\tbackground-color: rgba(255, 255, 255, 0.8);\n\t\t\tpadding: 10px;\n\t\t\tborder-radius: 5px;\n\t\t\tanimation: pulse 1.5s infinite;\n\t\t}\n\t\t@keyframes pulse {\n\t\t\t0% { transform: translate(-50%, -50%) scale(1); opacity: 1; }\n\t\t\t50% { transform: translate(-50%, -50%) scale(1.2); opacity: 0.7; }\n\t\t\t100% { transform: translate(-50%, -50%) scale(1); opacity: 1; }\n\t\t}\n\t`;\n\t\t\tdocument.head.appendChild(style);", nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("登录页面已加载完成")
			return nil
		}),
		// 加载登录按钮
		chromedp.WaitVisible("#login-btn", chromedp.ByID),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("登录按钮已加载完成")
			return nil
		}),
		// 点击登录按钮
		chromedp.Click("#login-btn", chromedp.ByID),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("点击登录按钮完成")
			return nil
		}),
		// 等待登录二维码加载
		chromedp.WaitVisible(".login-container .qrcode-img", chromedp.ByQuery),
		chromedp.ActionFunc(func(context.Context) error {
			log.Println("登录二维码加载完成")
			return nil
		}),

		chromedp.ActionFunc(func(context.Context) error {
			log.Println("点击登录完成")
			return nil
		}),
		chromedp.WaitVisible(".main-container .user .link-wrapper .channel", chromedp.ByQuery),
		chromedp.ActionFunc(func(context.Context) error {
			log.Println("登录成功")
			return nil
		}),
		// 获取所有 Cookie
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err := storage.GetCookies().Do(ctx)
			if err != nil {
				log.Println("Failed to get cookies: ", err.Error())
				return err
			}

			cookieBytes, err := json.Marshal(cookies)
			if err != nil {
				log.Println("Failed to marshal cookies: ", err.Error())
				return err
			}

			cookieStr := string(cookieBytes)
			log.Println("Cookies 保存成功: " + cookieStr)

			// 获取可执行文件所在目录
			exePath, err := os.Executable()
			if err != nil {
				log.Println("Failed to get executable path: ", err.Error())
				return err
			}
			exeDir := filepath.Dir(exePath)

			// 保存到同目录的txt文件
			cookieFile := filepath.Join(exeDir, CookiePath)
			err = os.WriteFile(cookieFile, []byte(cookieStr), 0644)
			if err != nil {
				log.Println("Failed to write cookie file: ", err.Error())
				return err
			}

			log.Println("Cookie已保存到文件: " + cookieFile)
			return nil
		}),
	)
	if err != nil {
		log.Println(fmt.Sprintf("%s: chromedp执行异常", LoginPage), err.Error())
		return errors.New("登录失败")
	}

	log.Println("登录完成")
	return nil
}

func Release(params v1.ReleaseParams) error {
	log.Println("开始发布")
	ok, err := CheckLogin()
	if err != nil {
		return err
	}
	if !ok {
		err := Login()
		if err != nil {
			return err
		}
	}

	downloadedImages, err := v1.DownloadImages(params)
	if err != nil {
		return err
	}
	if len(downloadedImages) == 0 {
		return fmt.Errorf("至少需要1张图片")
	}

	if len(params.Tags) == 0 {
		return fmt.Errorf("至少需要1个话题标签")
	}

	if len(params.Content) == 0 {
		return fmt.Errorf("内容不能为空")
	}

	if len(params.Title) == 0 {
		return fmt.Errorf("标题不能为空")
	}

	cookie, err := v1.ReadCookieFromFile(CookiePath)
	if err != nil {
		return err
	}

	ctx, cancel := GetNewChromedpContext()
	defer cancel()
	log.Println("上传图片，写如标题")
	err = chromedp.Run(ctx,
		v1.SetCookie(cookie),
		chromedp.Navigate(ReleasePage),
		chromedp.Sleep(2*time.Second),
		chromedp.WaitVisible(".upload-input", chromedp.ByQuery),
		chromedp.SetUploadFiles(".upload-input", downloadedImages),
		chromedp.SetValue("div.d-input input", params.Title),
		chromedp.WaitVisible(ContentSelector, chromedp.ByQuery),                   // 等待元素可见
		chromedp.Click(ContentSelector, chromedp.ByQuery),                         // 点击元素
		chromedp.SendKeys(ContentSelector, params.Content+"\n", chromedp.ByQuery), // 输入字符
	)
	if err != nil {
		log.Println("发布失败，chromedp执行异常: ", err.Error())
		return errors.New("发布失败，打开小红书失败，请检查客户端日志")
	}
	log.Println("写入内容完成")

	log.Println("开始写入话题")
	// 写入话题
	for _, tag := range params.Tags {
		// 使用 strings.HasPrefix 检查前缀
		if !strings.HasPrefix(tag, "#") {
			tag = "#" + tag
		}

		err := chromedp.Run(ctx,
			chromedp.SendKeys(ContentSelector, strings.TrimSpace(tag), chromedp.ByQuery), // 每次输入一个标签，添加空格分隔
			chromedp.WaitVisible("#creator-editor-topic-container", chromedp.ByID),
			chromedp.WaitVisible(".item.is-selected", chromedp.ByQuery), // 等待元素可见
			chromedp.Click(".item.is-selected", chromedp.ByQuery),       // 点击元素
		)
		if err != nil {
			return errors.New("发布失败，写入话题失败，请检查客户端日志")
		}
	}
	log.Println("写入话题完成")
	// 发布
	log.Println("开始发布")
	err = chromedp.Run(ctx,
		chromedp.Sleep(2*time.Second),
		chromedp.WaitVisible("button.publishBtn", chromedp.ByQuery), // 等待元素可见
		chromedp.Click("button.publishBtn", chromedp.ByQuery),       // 点击元素
		// 等待发布成功
		chromedp.WaitVisible(".success-container", chromedp.ByQuery),
	)
	if err != nil {
		return errors.New("发布失败，请检查客户端日志")
	}

	log.Println("发布成功")
	return nil
}
