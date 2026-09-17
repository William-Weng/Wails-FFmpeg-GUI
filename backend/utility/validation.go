package utility

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 驗證轉換選項是否已指定輸入影片，並確認輸入路徑存在、可存取且不是資料夾
func CheckFileExists(options ConversionOptions) error {

	if strings.TrimSpace(options.InputPath) == "" {
		return errors.New("請先選擇輸入影片")
	}

	return CheckInputFile(options.InputPath)
}

// 驗證指定路徑是否為存在且可存取的一般檔案
//
//   - 此函式只確認檔案存在且不是資料夾；不驗證副檔名、MIME type 或 FFmpeg 是否實際支援該媒體格式
func CheckInputFile(inputPath string) error {

	info, err := os.Stat(inputPath)

	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("輸入檔案不存在：%w", err)
		}
		return fmt.Errorf("無法讀取輸入檔案：%w", err)
	}

	if info.IsDir() {
		return errors.New("輸入路徑不是影片檔案，而是資料夾")
	}

	return nil
}

// 根據 FFmpeg 執行檔路徑推導同目錄的 ffprobe 路徑
//
//   - 若 ffmpegPath 為空，回傳 "ffprobe"，由系統 PATH 尋找
//   - 若檔名不是 "ffmpeg" 或 "ffmpeg.exe"，同樣回傳 "ffprobe"
//
// 範例：
//   - "/opt/homebrew/bin/ffmpeg" → "/opt/homebrew/bin/ffprobe"
//   - "C:\\tools\\ffmpeg.exe" → "C:\\tools\\ffprobe.exe"
func DeriveFFprobePath(ffmpegPath string) string {

	ffmpegPath = strings.TrimSpace(ffmpegPath)

	if ffmpegPath == "" {
		return "ffprobe"
	}

	ffmpegPath = filepath.Clean(ffmpegPath)
	dir := filepath.Dir(ffmpegPath)
	base := filepath.Base(ffmpegPath)

	switch base {
	case "ffmpeg":
		return filepath.Join(dir, "ffprobe")

	case "ffmpeg.exe":
		return filepath.Join(dir, "ffprobe.exe")

	default:
		return "ffprobe"
	}
}
