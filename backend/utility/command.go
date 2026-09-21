package utility

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var supportedMediaTypes = map[string]struct{}{
	"mp4":  {},
	"ts":   {},
	"mkv":  {},
	"mp3":  {},
	"aac":  {},
	"ogg":  {},
	"opus": {},
}

// 建立 FFmpeg 命令的標準輸入與標準錯誤輸出管道
//   - stdin 可用來向 FFmpeg 傳送控制指令，例如 "q\n"
//   - stderr 可用來讀取 FFmpeg 的進度資訊與錯誤訊息
//   - 如果 stderr 管道建立失敗，函式會關閉已建立的 stdin 管道，避免發生資源洩漏
//
// 回傳值:
//   - 依序為 stdin、stderr 和錯誤
func CreateFFmpegPipes(command *exec.Cmd) (io.WriteCloser, io.ReadCloser, error) {

	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("無法建立 FFmpeg stdin pipe：%w", err)
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, nil, fmt.Errorf("無法建立 FFmpeg stderr pipe：%w", err)
	}

	return stdin, stderr, nil
}

// 根據影片轉換選項和輸出檔案路徑，建立 FFmpeg 命令列所需的參數列表；此函式只負責組合參數，不會實際執行 FFmpeg
// 例如：
//   - ffmpeg -y -ss 00:01:00 -i input.mov -to 00:05:00 -vf scale=1280:720 -c:v libx264 -c:a aac output.mp4
//
// 參數：
//   - options：影片輸入路徑、時間範圍、編碼器和尺寸等轉換設定
//   - outputPath：轉換後的輸出檔案路徑
//
// 回傳：
//   - 可傳給 exec.Command 或 exec.CommandContext 的 FFmpeg 參數列表
func BuildFFmpegArguments(options ConversionOptions, outputPath string) ([]string, error) {

	args := []string{"-y"}

	startSeconds := 0.0

	if options.UseStart && options.StartTime != "" {
		seconds, err := ParseTimestamp(options.StartTime)
		if err != nil {
			return nil, fmt.Errorf("無效開始時間：%w", err)
		}

		startSeconds = seconds
		args = append(args, "-ss", options.StartTime)
	}

	if options.UseEnd && options.EndTime != "" {
		endSeconds, err := ParseTimestamp(options.EndTime)
		if err != nil {
			return nil, fmt.Errorf("無效結束時間：%w", err)
		}

		durationSeconds := endSeconds - startSeconds
		if durationSeconds <= 0 {
			return nil, fmt.Errorf("結束時間必須大於開始時間：開始=%s，結束=%s", options.StartTime, options.EndTime)
		}

		args = append(args, "-t", FormatTimestamp(durationSeconds))
	}

	args = append(args, "-i", options.InputPath)

	codec := strings.ToLower(strings.TrimSpace(options.VideoCodec))
	container := strings.ToLower(strings.TrimSpace(options.Container))

	switch container {
	case "mp3", "aac", "ogg", "opus":
		args = combineAudioArguments(container, args)
	default:
		args = combineVideoArguments(options, codec, args)
	}

	args = append(args, outputPath)
	return args, nil
}

// 根據指定的影片 codec 與轉換選項，組合 FFmpeg 的影片／音訊轉檔參數
//
//   - options：轉換設定，例如是否調整尺寸、目標寬度與高度
//   - codec：影片編碼模式，目前支援 "copy"、"h264"、"h265"
//   - args：目前已建立的 FFmpeg 參數；本函式會將新參數 append 到其中
//
// 回傳值：
//   - 已加入影片處理參數的 args
func combineVideoArguments(options ConversionOptions, codec string, args []string) []string {

	switch codec {
	case "copy":
		if options.UseSize {
			fmt.Println("警告：Copy 模式忽略尺寸設定")
		}

		args = append(
			args,
			"-map", "0:v:0",
			"-map", "0:a?",
			"-c", "copy",
		)

	case "h264":
		if options.UseSize && options.Width > 0 && options.Height > 0 {
			scale := fmt.Sprintf("scale=%d:%d", options.Width, options.Height)
			args = append(args, "-vf", scale)
		}

		args = append(args, "-c:v", "libx264", "-c:a", "aac")

	case "h265":
		if options.UseSize && options.Width > 0 && options.Height > 0 {
			scale := fmt.Sprintf("scale=%d:%d", options.Width, options.Height)
			args = append(args, "-vf", scale)
		}

		args = append(args, "-c:v", "libx265", "-c:a", "aac")

	default:
		args = append(args, "-c:v", "libx264", "-c:a", "aac")
	}

	return args
}

// 根據輸出音訊容器，將對應的 FFmpeg 音訊轉檔參數追加到 args
func combineAudioArguments(container string, args []string) []string {

	args = append(
		args,
		"-vn",
		"-map", "0:a:0?",
	)

	switch container {
	case "mp3":
		args = append(
			args,
			"-c:a", "libmp3lame",
			"-q:a", "2",
		)

	case "aac":
		args = append(
			args,
			"-c:a", "aac",
			"-b:a", "192k",
		)

	case "ogg":
		args = append(
			args,
			"-c:a", "libvorbis",
			"-q:a", "5",
		)

	case "opus":
		args = append(
			args,
			"-c:a", "libopus",
			"-b:a", "128k",
		)

	default:
		args = append(
			args,
			"-c:a", "aac",
			"-b:a", "192k",
		)
	}

	return args
}

// DetectMediaDuration 使用 ffprobe 取得媒體檔案的總時長
//   - ffprobePath：ffprobe 執檔的完整路徑
//   - inputPath：要讀取資訊的影音檔案路徑
//
// 回傳:
//   - duration：媒體總長度，單位為秒，例如 125.45 代表 2 分 5.45 秒。
//   - err：ffprobe 執行失敗、輸出無法轉為秒數，或取得非正時長時的錯誤。
func DetectMediaDuration(context context.Context, ffprobePath string, inputPath string) (duration float64, err error) {

	cmd := exec.CommandContext(
		context,
		ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	cmd = configureChildProcess(cmd)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe 執行失敗：%w", err)
	}

	duration, err = strconv.ParseFloat(strings.TrimSpace(string(output)), 64)

	if err != nil {
		return 0, fmt.Errorf("無法解析影片長度：%w", err)
	}

	if duration <= 0 {
		return 0, errors.New("影片長度必須大於 0")
	}

	return duration, nil
}

func configureChildProcess(cmd *exec.Cmd) *exec.Cmd {
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}

	return cmd
}

// DetectMediaType 使用 ffprobe 判斷輸入檔是否有視訊軌與音訊軌。
// 回傳：hasVideo, hasAudio, error
func DetectMediaType(ffprobePath, filePath string) (hasVideo bool, hasAudio bool, err error) {

	// 執行 ffprobe
	cmd := exec.Command(
		ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		filePath,
	)

	cmd = configureChildProcess(cmd)

	out, err := cmd.Output()
	if err != nil {
		return false, false, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result FFprobeResult
	if err := json.Unmarshal(out, &result); err != nil {
		return false, false, fmt.Errorf("parse ffprobe json failed: %w", err)
	}

	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "video":
			hasVideo = true
		case "audio":
			hasAudio = true
		}
		if hasVideo && hasAudio {
			break
		}
	}

	return hasVideo, hasAudio, nil
}

// 會根據輸入檔案路徑與容器格式，產生新的輸出檔案路徑；輸出檔案會保留原始檔案所在的資料夾與檔名，並在副檔名前加入時間戳記
//
// 例如：
//
//	inputPath: "/Users/me/Videos/movie.mov"
//	container: "mp4"
//
//	輸出："/Users/me/Videos/movie_20260915-102700.123.mp4"
func MakeOutputPath(inputPath string, container string) string {

	extension := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, extension)
	timestamp := time.Now().Format("20060102-150405.000")

	return fmt.Sprintf("%s_%s.%s", base, timestamp, container)
}

// 去除前後空白，並將英文字母轉成小寫；例如： ".MP4"  -> "mp4"
func NormalizeContainer(value string) string {

	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimLeft(value, ".")

	if _, found := supportedMediaTypes[value]; found {
		return value
	}

	return "mp4"
}
