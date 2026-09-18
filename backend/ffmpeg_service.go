package backend

import (
	"context"
	"errors"
	"ffmpeg-gui/backend/utility"
	util "ffmpeg-gui/backend/utility"
	"sync"

	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// FFmpegService 封裝 FFmpeg 轉換流程與狀態管理
type FFmpegService struct {
	App        *application.App // App 是 Wails 應用程式的實例；可用來和 Wails 的事件系統或應用程式生命週期互動
	mutex      sync.Mutex       // mutex 用來保護下方的共享狀態；由於 FFmpeg 可能在背景 goroutine 中執行，因此需要使用 Mutex 避免多個 goroutine 同時讀寫造成資料競爭
	running    bool             // running 表示目前是否有 FFmpeg 轉換工作正在執行
	cancelling bool             // cancelling 表示目前是否正在處理取消操作；例如：已收到取消要求，但 FFmpeg 程序尚未完全結束
	cancelled  bool             // cancelled 表示目前的轉換是否已被取消
	stdin      io.WriteCloser   // stdin 是 FFmpeg 程序的標準輸入；可透過寫入 "q" 要求 FFmpeg 優雅地結束
	cmd        *exec.Cmd        // cmd 儲存目前正在執行的 FFmpeg 外部程序；可透過它取得程序狀態，或呼叫 Process.Kill() 強制終止程序
}

var totalDuration = 0.0
var lastPercent = -1.0

// MARK: - Lifecycle Hooks
// ServiceStartup 是 Wails v3 的服務啟動生命週期勾子
//   - Wails 啟動應用程式時會呼叫此方法；這裡會取得目前的 Wails App 實例，讓 FFmpegService 後續可以透過 app 發送事件給前端
func (service *FFmpegService) ServiceStartup(ctx context.Context) error {
	service.App = application.Get()
	return nil
}

// ServiceShutdown 在 Wails App 關閉時停止仍在執行的 FFmpeg
func (service *FFmpegService) ServiceShutdown() error {

	service.mutex.Lock()

	cmd := service.cmd
	running := service.running

	service.mutex.Unlock()

	if running && cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	service.clearRunningState()
	return nil
}

// MARK: - Public Methods
// NewFFmpegService 建立並回傳一個新的 FFmpegService；回傳指標，讓後續可以在服務生命週期中修改其內部狀態
func NewFFmpegService() *FFmpegService {
	return &FFmpegService{}
}

//	使用 ffprobe 取得影片長度，單位為秒
//
// 回傳：
//   - duration：影片長度，例如 180.52 秒
//   - error：ffprobe 執行失敗或輸出格式無法解析
func (service *FFmpegService) GetVideoDuration(ffmpegPath string, inputPath string) (float64, error) {

	inputPath = strings.TrimSpace(inputPath)
	ffmpegPath = strings.TrimSpace(ffmpegPath)

	if inputPath == "" {
		return 0, errors.New("請先選擇輸入影片")
	}

	if err := util.CheckInputFile(inputPath); err != nil {
		return 0, err
	}

	ffprobePath := util.DeriveFFprobePath(ffmpegPath)

	command := exec.CommandContext(
		service.commandContext(),
		ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	output, err := command.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe 執行失敗：%w", err)
	}

	duration, err := strconv.ParseFloat(
		strings.TrimSpace(string(output)),
		64,
	)
	if err != nil {
		return 0, fmt.Errorf("無法解析影片長度：%w", err)
	}

	if duration <= 0 {
		return 0, errors.New("影片長度必須大於 0")
	}

	return duration, nil
}

// 啟動一次 FFmpeg 影片轉換工作
//   - 函式會先驗證輸入檔案，接著建立輸出路徑與 FFmpeg 參數，然後啟動 FFmpeg 並即時讀取其 stderr 輸出
//   - 不使用 -nostdin，才能在取消時對 FFmpeg 寫入 `q`
//   - 如果轉換期間收到取消要求，函式會等待 FFmpeg 嘗試正常結束
//   - 如果 FFmpeg 在指定時間內仍未結束，則會強制終止 FFmpeg 程序
//
// 回傳：
//   - 轉換成功時，回傳 ConversionResult 與 nil 錯誤
//   - 轉換失敗時，回傳空的 ConversionResult 與錯誤訊息
func (service *FFmpegService) StartConversion(options utility.ConversionOptions) (utility.ConversionResult, error) {

	if err := util.CheckFileExists(options); err != nil {
		return utility.ConversionResult{}, err
	}

	totalDuration, err := service.parseTotalDuration(options)

	if err != nil {
		return utility.ConversionResult{}, err
	}

	if err := service.beginConversion(); err != nil {
		return utility.ConversionResult{}, err
	}

	defer service.clearRunningState()

	command, outputPath, err := service.buildFFmpegCommand(options)

	if err != nil {
		return utility.ConversionResult{}, err
	}

	stdin, stderr, err := util.CreateFFmpegPipes(command)
	if err != nil {
		return utility.ConversionResult{}, err
	}

	service.mutex.Lock()
	service.stdin = stdin
	service.cmd = command
	service.mutex.Unlock()

	err = service.executeFFmpeg(command, stderr, totalDuration)

	service.mutex.Lock()
	wasCancelled := service.cancelled
	service.mutex.Unlock()

	return service.finalResult(outputPath, wasCancelled, err)
}

// 嘗試取消目前正在執行的 FFmpeg 轉換
//   - 函式會先檢查目前是否有可取消的轉換，再透過 FFmpeg 的標準輸入傳送 "q" 指令，讓 FFmpeg 有機會正常完成輸出檔案的收尾工作
//   - FFmpeg 讀取 stdin 時收到 q，會停止轉檔並正常寫 trailer
//   - 如果 FFmpeg 在 5 秒內仍未結束，則由背景程序強制終止 FFmpeg
//   - 以下任一條件成立時，就不能取消：
//     1. 沒有正在執行的轉換。
//     2. 已經正在取消中，避免重複送出取消指令。
//     3. 沒有 FFmpeg 的標準輸入管道。
//     4. 沒有目前執行中的 FFmpeg 命令。
//
// 回傳：
//   - true 表示取消要求已成功送出
//   - false 表示目前沒有可取消的轉換，或無法將取消指令寫入 FFmpeg 的標準輸入
func (service *FFmpegService) CancelConversion() bool {

	service.mutex.Lock()

	if !service.running || service.cancelling || service.stdin == nil || service.cmd == nil {
		service.mutex.Unlock()
		return false
	}

	service.cancelling = true
	service.cancelled = true

	stdin := service.stdin
	cmd := service.cmd

	service.mutex.Unlock()

	if _, err := io.WriteString(stdin, "q\n"); err != nil {
		service.mutex.Lock()
		service.cancelling = false
		service.cancelled = false
		service.mutex.Unlock()

		return false
	}

	service.emitOutput("\n\n----- 正在安全停止 FFmpeg -----")
	go service.forceKillIfNeeded(cmd, 5*time.Second)

	return true
}

// 取得目前 FFmpeg 轉換的狀態
//   - 函式會使用互斥鎖保護共享狀態，避免在其他 goroutine 修改狀態時讀取到不一致的資料
func (service *FFmpegService) GetConversionState() utility.ConversionState {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	return utility.ConversionState{
		Running:    service.running,
		Cancelling: service.cancelling,
		Cancelled:  service.cancelled,
	}
}

// MARK: - Private Methods
// 等待指定的時間後，檢查 FFmpeg 是否仍在執行
//   - 如果指定的命令仍是目前服務管理中的命令，就強制終止該 FFmpeg 程序
//   - cmd 必須是目前啟動的 FFmpeg 命令，timeout 則是等待優雅結束的最長時間
func (service *FFmpegService) forceKillIfNeeded(cmd *exec.Cmd, timeout time.Duration) {

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	<-timer.C

	service.mutex.Lock()
	stillRunning := service.running && service.cmd == cmd
	service.mutex.Unlock()

	if !stillRunning {
		return
	}

	service.emitOutput("----- FFmpeg 未在 5 秒內停止，正在強制結束 -----")

	_ = cmd.Process.Kill()
}

// 清除目前 FFmpeg 工作的執行狀態
//   - 函式會關閉 FFmpeg 的標準輸入，重設轉換狀態，並清除目前保存的程序與輸入串流參考
//   - 使用互斥鎖確保清理共享狀態時，不會和其他 goroutine 同時存取
func (service *FFmpegService) clearRunningState() {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	if service.stdin != nil {
		_ = service.stdin.Close()
	}

	service.running = false
	service.cancelling = false
	service.cancelled = false
	service.stdin = nil
	service.cmd = nil
}

// 輸出一行 FFmpeg 訊息
//   - 沒有 Wails App 時，函式會使用 fmt.Println 將訊息輸出到終端機，適合在測試或獨立執行服務時使用
//   - 有 Wails App 時，函式會發送 "ffmpeg:output" 事件，並將訊息包裝成 FFmpegOutput 結構，讓前端可以接收
func (service *FFmpegService) emitOutput(line string) {

	if service.App == nil {
		fmt.Println(line)
		return
	}

	service.App.Event.Emit(util.FFmpegEventOutput.String(), utility.FFmpegOutput{Line: line})
}

// 發送 FFmpeg 目前的轉換進度
//   - 沒有 Wails App 時，進度會輸出至終端機，方便測試或以獨立模式執行
//   - 有 Wails App 時，會透過 ffmpeg:progress 事件將 FFmpegProgress
//   - 資料傳送給前端，用於更新進度條、處理時間與百分比
func (service *FFmpegService) emitProgress(progress utility.FFmpegProgress) {

	if service.App == nil {
		fmt.Printf("FFmpeg progress: %.1f%% (%.2f / %.2f sec)\n", progress.Percent, progress.CurrentSeconds, progress.TotalSeconds)
		return
	}

	service.App.Event.Emit(util.FFmpegEventProgress.String(), progress)
}

// 發送 FFmpeg 成功完成轉換的事件
//   - 沒有 Wails App 時，完成結果會輸出至終端機
//   - 有 Wails App 時，會透過 ffmpeg:completed 事件將輸出檔案路徑
//   - 包裝為 FFmpegCompleted 資料傳送給前端
func (service *FFmpegService) emitCompleted(outputPath string) {

	if service.App == nil {
		fmt.Printf("FFmpeg completed: %s\n", outputPath)
		return
	}

	service.App.Event.Emit(util.FFmpegEventCompleted.String(), utility.FFmpegCompleted{OutputPath: outputPath})
}

// 發送 FFmpeg 轉換失敗的事件
//   - 沒有 Wails App 時，錯誤會輸出至終端機
//   - 有 Wails App 時，會透過 ffmpeg:failed 事件將錯誤訊息包裝為
//   - FFmpegFailed 資料傳送給前端
func (service *FFmpegService) emitFailed(err error) {

	if err == nil {
		return
	}

	if service.App == nil {
		fmt.Printf("FFmpeg failed: %v\n", err)
		return
	}

	service.App.Event.Emit(util.FFmpegEventFailed.String(), utility.FFmpegFailed{Message: err.Error()})
}

// 從 reader 逐字元讀取 FFmpeg 的輸出內容，每次只從 reader 讀取一個字元
//   - 當讀到換行字元（\n）或回車字元（\r）時，就將目前累積的內容傳送給前端
//   - 如果輸入串流結束時仍有尚未傳送的內容，函式會在收到 io.EOF 時一併送出最後一行
func (service *FFmpegService) streamFFmpegOutput(reader io.Reader, totalDuration float64) {

	buffer := make([]byte, 1)
	var line strings.Builder

	lastPercent := -1.0

	emit := func(suffix byte) {
		if line.Len() == 0 {
			return
		}

		output := line.String()

		service.emitOutput(output + string(suffix))

		lastPercent = service.updateProgress(output, suffix, totalDuration, lastPercent)
		line.Reset()
	}

	for {
		count, err := reader.Read(buffer)

		if count > 0 {
			switch buffer[0] {
			case '\r':
				emit('\r')

			case '\n':
				emit('\n')

			default:
				line.WriteByte(buffer[0])
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				emit(0)
			}

			return
		}
	}
}

// 檢查目前是否已有 FFmpeg 轉換正在執行，並在沒有其他轉換時初始化新的轉換狀態
//   - 如果已有轉換正在執行，回傳錯誤
//   - 成功時將 running 設為 true，並清除前一次工作的取消狀態、stdin 管道和命令參考
func (service *FFmpegService) beginConversion() error {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	if service.running {
		return errors.New("已有一個 FFmpeg 轉換正在執行")
	}

	service.running = true
	service.cancelling = false
	service.cancelled = false
	service.stdin = nil
	service.cmd = nil

	return nil
}

// 取得用來執行 FFmpeg 命令的 Context；如果 FFmpegService 沒有關聯 Wails App，則回傳 context.Background()
func (service *FFmpegService) commandContext() context.Context {

	if service.App == nil {
		return context.Background()
	}

	return service.App.Context()
}

// 根據轉換選項建立尚未啟動的 FFmpeg 命令
//   - 此方法會標準化輸出容器格式、建立輸出檔案路徑、產生 FFmpeg
//   - 命令列參數，並使用 ConversionOptions.FFmpegPath 建立 *exec.Cmd。
//   - 若 FFmpegPath 為空字串，則使用系統 PATH 中的 "ffmpeg"。
//
// 回傳值依序為：
//   - *exec.Cmd：尚未啟動的 FFmpeg 命令
//   - string：預計產生的輸出影片路徑
//   - error：建立參數或命令時發生的錯誤
//
// 此方法只建立命令，不會執行 FFmpeg；呼叫端需使用 command.Start()、command.Run()、command.Wait() 或 command.Output() 啟動或等待外部程序。
func (service *FFmpegService) buildFFmpegCommand(options utility.ConversionOptions) (*exec.Cmd, string, error) {

	container := util.NormalizeContainer(options.Container)
	outputPath := util.MakeOutputPath(options.InputPath, container)
	args, err := util.BuildFFmpegArguments(options, outputPath)

	if err != nil {
		return nil, "", err
	}

	ffmpegPath := strings.TrimSpace(options.FFmpegPath)
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	command := exec.CommandContext(service.commandContext(), ffmpegPath, args...)
	return command, outputPath, nil
}

// 啟動並等待 FFmpeg 命令完成
//   - 函式會在背景 goroutine 中持續讀取 FFmpeg 的 stderr，同時在目前的 goroutine 中等待 FFmpeg 程序結束
//   - 回傳值為 FFmpeg 的執行錯誤；nil 表示命令成功完成
func (service *FFmpegService) executeFFmpeg(command *exec.Cmd, stderr io.ReadCloser, totalDuration float64) error {

	service.emitOutput(fmt.Sprintf("\n\n----- FFmpeg 指令 -----\n%s\n", command.String()))
	service.emitOutput("\n----- FFmpeg 輸出 -----")

	if err := command.Start(); err != nil {
		_ = stderr.Close()
		return fmt.Errorf("無法啟動 FFmpeg：%w", err)
	}

	outputDone := make(chan struct{})

	go func() {
		defer close(outputDone)
		service.streamFFmpegOutput(stderr, totalDuration)
	}()

	err := command.Wait()

	<-outputDone

	return err
}

// parseTotalDuration 計算本次 FFmpeg 轉換預期輸出的總長度，單位為秒
//   - 此長度用於計算轉換進度百分比，而不是用於取得來源影片 metadata
//   - 若未設定剪輯範圍，回傳來源影片完整長度
//   - 使用者有設定開始時間：輸出從該時間開始
//   - 使用者有設定結束時間：輸出在來源時間軸的該位置停止
//   - 避免使用者輸入超出影片長度的 end time
func (service *FFmpegService) parseTotalDuration(options utility.ConversionOptions) (float64, error) {

	sourceDuration, err := service.GetVideoDuration(options.FFmpegPath, options.InputPath)

	if err != nil {
		return 0, err
	}

	startSeconds := 0.0
	endSeconds := sourceDuration

	if options.UseStart && strings.TrimSpace(options.StartTime) != "" {

		startSeconds, err = util.ParseTimestamp(options.StartTime)

		if err != nil {
			return 0, fmt.Errorf("解析開始時間失敗：%w", err)
		}
	}

	if options.UseEnd && strings.TrimSpace(options.EndTime) != "" {
		endSeconds, err = util.ParseTimestamp(options.EndTime)
		if err != nil {
			return 0, fmt.Errorf("解析結束時間失敗：%w", err)
		}
	}

	if endSeconds > sourceDuration {
		endSeconds = sourceDuration
	}

	if startSeconds >= sourceDuration {
		return 0, fmt.Errorf("開始時間必須小於影片總長度：開始=%s，影片長度=%s", options.StartTime, util.FormatTimestamp(sourceDuration))
	}

	if endSeconds <= startSeconds {
		return 0, fmt.Errorf("結束時間必須大於開始時間：開始=%s，結束=%s", options.StartTime, options.EndTime)
	}

	return endSeconds - startSeconds, nil
}

// updateProgress 從 FFmpeg 的即時輸出狀態列更新轉換進度
//   - FFmpeg 的傳統進度列通常以 carriage return（\r）結尾；因此只處理 suffix 為 \r 的輸出，避免一般 metadata、warning、summary 或 error log 也被當作進度解析
//   - lastPercent 是上一次已發送給前端的百分比。若本次進度與前次差距
//   - 小於 0.1%，不發送事件並回傳原值，以減少 Wails event 與 UI 重繪
//   - 回傳值必須由呼叫端保存，作為下一次的 lastPercent
func (service *FFmpegService) updateProgress(output string, suffix byte, totalDuration float64, lastPercent float64) float64 {

	if suffix != '\r' {
		return lastPercent
	}

	currentSeconds, found := util.ParseFFmpegTimeFromOutput(output)
	if !found {
		return lastPercent
	}

	percent := util.CalculateProgress(currentSeconds, totalDuration)

	// 避免同一百分比或差異極小的進度反覆觸發前端重繪
	if percent-lastPercent < 0.1 {
		return lastPercent
	}

	service.emitProgress(utility.FFmpegProgress{CurrentSeconds: currentSeconds, TotalSeconds: totalDuration, Percent: percent})
	return percent
}

// 根據 FFmpeg 的執行錯誤與取消狀態，建立最終轉換結果
//   - err 不為 nil 時，表示 FFmpeg 執行失敗
//   - wasCancelled 為 true 時，表示使用者曾要求取消轉換
//
// 回傳結果分為三種情況：
//   - 取消且 FFmpeg 發生錯誤：回傳取消錯誤
//   - 未取消但 FFmpeg 發生錯誤：回傳 FFmpeg 執行錯誤
//   - FFmpeg 正常結束：回傳輸出檔案路徑與成功訊息
func (service *FFmpegService) finalResult(outputPath string, wasCancelled bool, err error) (utility.ConversionResult, error) {

	if err != nil {
		if wasCancelled {
			return utility.ConversionResult{}, fmt.Errorf("轉換已取消；FFmpeg 未能正常收尾，輸出檔可能不完整：%s", outputPath)
		}
		return utility.ConversionResult{}, fmt.Errorf("FFmpeg 執行失敗：%w", err)
	}

	if wasCancelled {
		return utility.ConversionResult{OutputPath: outputPath, Message: "\n----- 轉換已取消 -----\n已保留 FFmpeg 正常收尾的部分輸出：\n" + outputPath}, nil
	}

	service.emitCompleted(outputPath)
	return utility.ConversionResult{OutputPath: outputPath, Message: "\n----- 轉換完成 -----\n" + outputPath}, nil
}
