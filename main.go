package main

import (
	"embed"

	"ffmpeg-gui/backend"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	var app *application.App

	// 1. 初始化服務實體
	ffmpegService := backend.NewFFmpegService()

	// 2. 建立 Wails v3 應用程式並綁定服務
	app = application.New(application.Options{
		Name: "影片格式轉換器",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Services: []application.Service{
			application.NewService(ffmpegService),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	ffmpegService.App = app // 將 app 實例傳遞給服務

	// 3. 核心修正：使用 v3 最新 API 建立視窗
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "影片格式轉換器",
		Width:          800,
		Height:         600,
		MinWidth:       600,
		MinHeight:      500,
		EnableFileDrop: true, // 開啟檔案拖放
		URL:            "/",
	}).OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		// 當使用者拖放檔案進視窗時觸發
		ctx := event.Context()
		files := ctx.DroppedFiles()

		if len(files) > 0 {
			// 發送全域事件給前端
			application.Get().Event.Emit("video-file-dropped", files)
		}
	})

	// 4. 啟動應用程式
	err := app.Run()
	if err != nil {
		println(err.Error())
	}
}
