export enum WailsEventType {
  VideoFileDropped = "video-file-dropped",
}

export enum FFmpegEventType {
  Output = "ffmpeg:output",
  Progress = "ffmpeg:progress",
  Started = "ffmpeg:started",
  Completed = "ffmpeg:completed",
  Cancelled = "ffmpeg:cancelled",
  Failed = "ffmpeg:failed",
}

export interface FFmpegProgress {
  currentSeconds: number;
  totalSeconds: number;
  percent: number;
}