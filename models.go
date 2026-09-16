package main

import (
	"io"
	"os/exec"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// FFmpegOutput 表示 FFmpeg 輸出的單行文字；通常會將 FFmpeg 的標準輸出或錯誤輸出逐行傳送給前端
type FFmpegOutput struct {
	Line string `json:"line"` // Line 儲存 FFmpeg 輸出的內容；`json:"line"` 表示轉換成 JSON 時，欄位名稱會是 "line"
}

// ConversionOptions 定義影片轉換時所需的設定
type ConversionOptions struct {
	InputPath  string `json:"inputPath"`  // InputPath 是輸入影片的檔案路徑
	FFmpegPath string `json:"ffmpegPath"` // FFmpegPath 是 FFmpeg 執行檔的路徑；例如："/usr/local/bin/ffmpeg"
	Container  string `json:"container"`  // Container 是輸出的容器格式；例如："mp4"、"mov" 或 "mkv"
	VideoCodec string `json:"videoCodec"` // VideoCodec 是使用的影片編碼器；例如："libx264"、"libx265" 或 "libvpx-vp9"
	StartTime  string `json:"startTime"`  // StartTime 是剪輯或轉換的開始時間；通常使用 FFmpeg 支援的時間格式，例如："00:01:30"
	EndTime    string `json:"endTime"`    // EndTime 是剪輯或轉換的結束時間；通常使用 FFmpeg 支援的時間格式，例如："00:05:00"
	Width      int    `json:"width"`      // Width 是輸出影片的寬度，單位為像素
	Height     int    `json:"height"`     // Height 是輸出影片的高度，單位為像素
	UseStart   bool   `json:"useStart"`   // UseStart 表示是否套用 StartTime
	UseEnd     bool   `json:"useEnd"`     // UseEnd 表示是否套用 EndTime
	UseSize    bool   `json:"useSize"`    // UseSize 表示是否套用 Width 和 Height 進行縮放
}

// ConversionResult 表示影片轉換完成後的結果
type ConversionResult struct {
	OutputPath string `json:"outputPath"` // OutputPath 是轉換後輸出檔案的路徑
	Message    string `json:"message"`    // Message 是提供給前端或使用者閱讀的結果訊息
}

// ConversionState 表示目前影片轉換工作的狀態；此結構通常會序列化成 JSON 後傳送給前端
type ConversionState struct {
	Running    bool `json:"running"`    // Running 表示目前是否正在執行轉換
	Cancelling bool `json:"cancelling"` // Cancelling 表示是否正在等待取消操作完成
	Cancelled  bool `json:"cancelled"`  // Cancelled 表示轉換是否已經被取消
}

// FFmpegService 封裝 FFmpeg 轉換流程與狀態管理
type FFmpegService struct {
	app        *application.App // app 是 Wails 應用程式的實例；可用來和 Wails 的事件系統或應用程式生命週期互動
	mutex      sync.Mutex       // mu 用來保護下方的共享狀態；由於 FFmpeg 可能在背景 goroutine 中執行，因此需要使用 Mutex 避免多個 goroutine 同時讀寫造成資料競爭
	running    bool             // running 表示目前是否有 FFmpeg 轉換工作正在執行
	cancelling bool             // cancelling 表示目前是否正在處理取消操作；例如：已收到取消要求，但 FFmpeg 程序尚未完全結束
	cancelled  bool             // cancelled 表示目前的轉換是否已被取消
	stdin      io.WriteCloser   // stdin 是 FFmpeg 程序的標準輸入；可透過寫入 "q" 要求 FFmpeg 優雅地結束
	cmd        *exec.Cmd        // cmd 儲存目前正在執行的 FFmpeg 外部程序；可透過它取得程序狀態，或呼叫 Process.Kill() 強制終止程序
}
