package utility

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var ffmpegTimePattern = regexp.MustCompile(
	`(?:^|\s)time=\s*([0-9:.]+)`,
)

// 根據目前已處理秒數與總秒數計算百分比
//
// - 當 totalDuration 小於或等於 0 時回傳 0
// - 回傳值會限制在 0 到 100，避免 FFmpeg timestamp、封裝 metadata 或浮點數誤差造成 UI 顯示負數或超過 100%
func CalculateProgress(currentSeconds float64, totalDuration float64) float64 {

	if totalDuration <= 0 {
		return 0
	}

	percent := currentSeconds / totalDuration * 100

	if percent < 0 {
		return 0
	}

	if percent > 100 {
		return 100
	}

	return percent
}

// 將 HH:MM:SS 或 HH:MM:SS.xx 格式的時間轉換為秒數
//
// 此函式同時用於：
//   - 使用者輸入的開始與結束時間
//   - FFmpeg status line 中的 time=00:00:51.54
//
// 時、分、秒皆不可為負數。回傳結果保留秒數的小數部分
func ParseTimestamp(value string) (float64, error) {
	value = strings.TrimSpace(value)

	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("無效時間格式 %q，預期 HH:MM:SS 或 HH:MM:SS.xx", value)
	}

	hours, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("解析小時失敗：%w", err)
	}

	minutes, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, fmt.Errorf("解析分鐘失敗：%w", err)
	}

	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, fmt.Errorf("解析秒數失敗：%w", err)
	}

	if hours < 0 || minutes < 0 || seconds < 0 {
		return 0, fmt.Errorf("時間不可為負數：%q", value)
	}

	// 可選但建議：防止 00:99:99 這類不符合時鐘格式的輸入。
	if minutes >= 60 || seconds >= 60 {
		return 0, fmt.Errorf("分鐘與秒數必須小於 60：%q", value)
	}

	return hours*3600 + minutes*60 + seconds, nil
}

// 將秒數格式化為 FFmpeg 可使用的 HH:MM:SS.xx 時間字串
//
//   - 負數會視為 0
//   - 秒數保留兩位小數，適合用於 -t 等時間參數
//
// 範例：
//   - 0 → "00:00:00.00"
//   - 51.54 → "00:00:51.54"
//   - 764 → "00:12:44.00"
func FormatTimestamp(totalSeconds float64) string {

	if totalSeconds < 0 {
		totalSeconds = 0
	}

	hours := int(totalSeconds) / 3600
	minutes := (int(totalSeconds) % 3600) / 60
	seconds := totalSeconds - float64(hours*3600+minutes*60)

	return fmt.Sprintf("%02d:%02d:%05.2f", hours, minutes, seconds)
}

// 從一筆 FFmpeg 狀態列擷取 time= 欄位並轉換為秒數
//
// 範例輸入：
//
//	"frame=1240 fps=388 time=00:00:51.54 bitrate=564.0kbits/s"
//
// 成功時回傳：
//
//	51.54, true
//
// 若該輸出不是進度列、找不到 time=，或時間格式無法解析，則回傳 0, false
func ParseFFmpegTimeFromOutput(line string) (float64, bool) {

	matches := ffmpegTimePattern.FindStringSubmatch(line)

	if len(matches) != 2 {
		return 0, false
	}

	seconds, err := ParseTimestamp(matches[1])
	if err != nil {
		return 0, false
	}

	return seconds, true
}
