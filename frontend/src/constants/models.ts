/**
 * FFmpeg 即時輸出資料
 *
 * - 由後端透過 `ffmpeg:output` 事件發送
 */
export interface FFmpegOutput {
  line: string;             // 一筆 FFmpeg 原始 stderr 輸出文字
}

/**
 * FFmpeg 轉換進度資料
 *
 * - 由後端透過 `ffmpeg:progress` 事件發送，前端可用來更新進度條、百分比與目前處理時間
 */
export interface FFmpegProgress {
    currentSeconds: number; // FFmpeg 目前已處理到的輸出時間，單位為秒
    totalSeconds: number;   // 來源影片或預期輸出的總長度，單位為秒
    percent: number;        // 轉換完成百分比，正常範圍為 0～100
}