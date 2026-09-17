package backend

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 建立 FFmpeg 命令的標準輸入與標準錯誤輸出管道
//   - stdin 可用來向 FFmpeg 傳送控制指令，例如 "q\n"
//   - stderr 可用來讀取 FFmpeg 的進度資訊與錯誤訊息
//   - 如果 stderr 管道建立失敗，函式會關閉已建立的 stdin 管道，避免發生資源洩漏
//
// 回傳值:
//   - 依序為 stdin、stderr 和錯誤
func createFFmpegPipes(command *exec.Cmd) (io.WriteCloser, io.ReadCloser, error) {

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
func buildFFmpegArguments(options ConversionOptions, outputPath string) ([]string, error) {

	args := []string{"-y"}

	startSeconds := 0.0

	if options.UseStart && options.StartTime != "" {
		seconds, err := parseTimestamp(options.StartTime)
		if err != nil {
			return nil, fmt.Errorf("無效開始時間：%w", err)
		}

		startSeconds = seconds
		args = append(args, "-ss", options.StartTime)
	}

	if options.UseEnd && options.EndTime != "" {
		endSeconds, err := parseTimestamp(options.EndTime)
		if err != nil {
			return nil, fmt.Errorf("無效結束時間：%w", err)
		}

		durationSeconds := endSeconds - startSeconds
		if durationSeconds <= 0 {
			return nil, fmt.Errorf(
				"結束時間必須大於開始時間：開始=%s，結束=%s",
				options.StartTime,
				options.EndTime,
			)
		}

		args = append(args, "-t", formatTimestamp(durationSeconds))
	}

	args = append(args, "-i", options.InputPath)

	codec := strings.ToLower(strings.TrimSpace(options.VideoCodec))

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

	args = append(args, outputPath)

	return args, nil
}

// 會根據輸入檔案路徑與容器格式，產生新的輸出檔案路徑；輸出檔案會保留原始檔案所在的資料夾與檔名，並在副檔名前加入時間戳記
//
// 例如：
//
//	inputPath: "/Users/me/Videos/movie.mov"
//	container: "mp4"
//
//	輸出："/Users/me/Videos/movie_20260915-102700.123.mp4"
func makeOutputPath(inputPath string, container string) string {

	extension := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, extension)
	timestamp := time.Now().Format("20060102-150405.000")

	return fmt.Sprintf("%s_%s.%s", base, timestamp, container)
}

// 去除前後空白，並將英文字母轉成小寫；例如： ".MP4"  -> "mp4"
func normalizeContainer(value string) string {

	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, ".")

	switch value {
	case "mkv", "ts":
		return value
	default:
		return "mp4"
	}
}
