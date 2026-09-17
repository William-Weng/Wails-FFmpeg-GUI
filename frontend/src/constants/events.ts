/**
 * Wails 應用程式層級事件名稱。
 *
 * - 用於不屬於特定功能模組的事件，例如視窗拖放檔案
 */
export enum WailsEventType {
  VideoFileDropped = "video-file-dropped",  // 使用者將影片檔案拖放至應用程式視窗時觸發
}

/**
 * FFmpeg 影片處理事件名稱。
 *
 * - 所有 FFmpeg 相關事件皆使用 `ffmpeg:` 作為 namespace，讓前端訂閱、除錯與事件搜尋時可與其他 Wails 事件區分
 */
export enum FFmpegEventType {
  Output = "ffmpeg:output",                 // 收到一筆 FFmpeg 原始輸出訊息，通常來自 stderr
  Progress = "ffmpeg:progress",             // 收到已結構化的轉換進度資料，例如目前秒數、總秒數與百分比
  Started = "ffmpeg:started",               // FFmpeg 開始執行轉換工作
  Completed = "ffmpeg:completed",           // FFmpeg 已成功完成轉換，輸出檔案已建立
  Cancelled = "ffmpeg:cancelled",           // FFmpeg 工作被使用者主動取消
  Failed = "ffmpeg:failed",                 // FFmpeg 無法啟動或轉換過程發生錯誤
}

