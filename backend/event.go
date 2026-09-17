package backend

type FFmpegEvent string

const (
	FFmpegEventOutput    FFmpegEvent = "ffmpeg:output"
	FFmpegEventProgress  FFmpegEvent = "ffmpeg:progress"
	FFmpegEventStarted   FFmpegEvent = "ffmpeg:started"
	FFmpegEventCompleted FFmpegEvent = "ffmpeg:completed"
	FFmpegEventCancelled FFmpegEvent = "ffmpeg:cancelled"
	FFmpegEventFailed    FFmpegEvent = "ffmpeg:failed"
)

func (event FFmpegEvent) String() string {
	return string(event)
}
