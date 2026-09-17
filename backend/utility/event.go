package utility

// FFmpegEvent 表示 FFmpeg 服務向前端發送的事件名稱
type FFmpegEvent string

const (
	FFmpegEventOutput    FFmpegEvent = "ffmpeg:output"    // FFmpegEventOutput 表示收到一筆 FFmpeg 原始輸出文字
	FFmpegEventProgress  FFmpegEvent = "ffmpeg:progress"  // FFmpegEventProgress 表示 FFmpeg 轉換進度已更新
	FFmpegEventStarted   FFmpegEvent = "ffmpeg:started"   // FFmpegEventStarted 表示 FFmpeg 已開始執行轉換
	FFmpegEventCompleted FFmpegEvent = "ffmpeg:completed" // FFmpegEventCompleted 表示 FFmpeg 已成功完成轉換
	FFmpegEventCancelled FFmpegEvent = "ffmpeg:cancelled" // FFmpegEventCancelled 表示 FFmpeg 轉換已由使用者取消
	FFmpegEventFailed    FFmpegEvent = "ffmpeg:failed"    // FFmpegEventFailed 表示 FFmpeg 無法啟動或轉換失敗
)

// String 回傳 FFmpegEvent 對應的事件名稱字串
func (event FFmpegEvent) String() string {
	return string(event)
}
