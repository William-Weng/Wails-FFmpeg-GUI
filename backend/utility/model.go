package utility

// FFmpegOutput 表示 FFmpeg 輸出的單行文字；通常會將 FFmpeg 的標準輸出或錯誤輸出逐行傳送給前端
type FFmpegOutput struct {
	Line string `json:"line"` // Line 儲存 FFmpeg 輸出的內容；`json:"line"` 表示轉換成 JSON 時，欄位名稱會是 "line"
}

// FFmpegProgress 表示 FFmpeg 影片轉換的即時進度
type FFmpegProgress struct {
	CurrentSeconds float64 `json:"currentSeconds"` // CurrentSeconds 是 FFmpeg 目前已處理到的輸出時間，單位為秒
	TotalSeconds   float64 `json:"totalSeconds"`   // TotalSeconds 是來源影片或預期輸出的總長度，單位為秒
	Percent        float64 `json:"percent"`        // Percent 是目前轉換進度百分比，正常範圍為 0 到 100
}

// FFmpegCompleted 表示 FFmpeg 已成功完成影片轉換
type FFmpegCompleted struct {
	OutputPath string `json:"outputPath"` // OutputPath 是成功建立的輸出影片完整路徑
}

// FFmpegFailed 表示 FFmpeg 無法啟動或影片轉換過程失敗
type FFmpegFailed struct {
	Message string `json:"message"` // Message 是可供前端顯示或寫入 log 的錯誤訊息
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
