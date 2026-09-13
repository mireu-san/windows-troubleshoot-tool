package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if handleAppUpdate() {
		return
	}
	app := NewApp()
	webviewMessages := windows.DefaultMessages()
	webviewMessages.InstallationRequired = "이 앱을 실행하려면 Microsoft Edge WebView2가 필요합니다. "
	webviewMessages.UpdateRequired = "Microsoft Edge WebView2 업데이트가 필요합니다. "
	webviewMessages.MissingRequirements = "필수 구성 요소 확인"
	webviewMessages.Webview2NotInstalled = "WebView2가 설치되지 않아 앱을 실행할 수 없습니다."
	webviewMessages.Error = "오류"
	webviewMessages.FailedToInstall = "Microsoft 페이지에서 WebView2를 설치한 뒤 이 앱을 다시 실행해 주세요."
	webviewMessages.DownloadPage = "이 앱을 실행하려면 Microsoft Edge WebView2를 먼저 설치해야 합니다. 확인을 누르면 Microsoft 설치 페이지를 엽니다. 설치한 뒤 이 앱을 다시 실행해 주세요. 필요한 최소 버전: "
	webviewMessages.PressOKToInstall = "확인을 누르면 자동으로 설치합니다."
	webviewMessages.ContactAdmin = "WebView2 설치를 위해 시스템 관리자에게 문의해 주세요."
	webviewMessages.InvalidFixedWebview2 = "지정된 WebView2 경로를 사용할 수 없습니다."
	webviewMessages.WebView2ProcessCrash = "화면 구성 요소가 종료되었습니다. 앱을 다시 실행해 주세요."
	err := wails.Run(&options.App{
		Title:     "Windows 시스템 복구 도우미",
		Width:     960,
		Height:    680,
		MinWidth:  360,
		MinHeight: 420,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 243, G: 246, B: 251, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		Bind:             []interface{}{app},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Messages:             webviewMessages,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
