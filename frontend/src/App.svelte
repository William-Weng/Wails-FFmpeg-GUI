<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Events, Dialogs } from "@wailsio/runtime";

  import { WailsEventType, FFmpegEventType } from "./constants/events";
  import { Containers, VideoCodecs } from "./constants/media";
  import type { FFmpegOutput, FFmpegProgress } from "./constants/models";
  import type { Container, VideoCodec } from "./constants/media";

  import FFmpegPathInput from "./components/FfmpegPathInput.svelte";
  import FilePathInput from "./components/FilePathInput.svelte";
  import OutputFormatSelect from "./components/OutputFormatSelect.svelte";
  import VideoCodecSelect from "./components/VideoCodecSelect.svelte";
  import ConvertButton from "./components/ConvertButton.svelte";
  import TimeOptionRow from "./components/TimeOptionRow.svelte";
  import SizeOptionRow from "./components/SizeOptionRow.svelte";

  import { StartConversion, CancelConversion, GetVideoDuration, DetectMediaType } from "../bindings/ffmpeg-gui/backend/ffmpegservice";
  import { Print } from "../bindings/ffmpeg-gui/backend/tools";
  import { GetFFmpegPath, SetFFmpegPath } from "../bindings/ffmpeg-gui/backend/configservice";

  type DialogType = "info" | "error" | "warning";

  let ffmpegPath = "ffmpeg";
  let inputPath = "";
  let container: Container = Containers.MP4
  let videoCodec: VideoCodec = VideoCodecs.Copy

  let enableStart = false;
  let enableEnd = false;
  let enableSize = false;
  let converting = false;
  let cancelling = false;

  let isVideo = false
  let videoDuration = 0;
  let videoDurationText = "";
  let startTime = "00:00:00";
  let endTime = "00:01:00";
  let width = 1920;
  let height = 1080;

  let progress = 0;

  let logText = "請拖放影片檔案到上方區域";
  let logElement: HTMLElement | null = null;

  async function saveFFmpegPath(path: string) {
    try {
      await SetFFmpegPath(path);
      console.log("FFmpeg 路徑已儲存");
    } catch (error) {
      console.error("儲存 FFmpeg 路徑失敗", error);
    }
  }

  async function loadFFmpegPath() {
  try {
    const path = await GetFFmpegPath();

    if (path) {
      ffmpegPath = path;
    }
  } catch (error) {
    console.error("讀取 FFmpeg 路徑失敗", error);
  }
}

  /**
   * 元件掛載時註冊 Wails 事件監聽器：
   *
   * 1. "video-file-dropped"
   *    - 處理使用者拖放影片檔案後的資料。
   *
   * 2. "ffmpeg:output"
   *    - 接收 FFmpeg stderr 的即時文字輸出。
   *    - 一般以 `\n` 結束的訊息會追加為新 log 行；
   *      以 `\r` 結束的進度列則覆寫目前最後一行。
   *
   * 3. "ffmpeg:progress"
   *    - 接收後端依 FFmpeg `time=` 與影片總長度計算的結構化進度資料。
   *    - 用於更新 progress element 與百分比文字。
   *
   * onMount 回傳的 cleanup function 會在元件卸載時執行，
   * 取消所有事件訂閱，避免重新掛載元件後產生重複 listener。
   */
  onMount(() => {
    const unsubscribeDrop = Events.On(WailsEventType.VideoFileDropped, (event) => {
      videoFileDroppedAction(event);
    });

    const unsubscribeFFmpegOutput = Events.On(FFmpegEventType.Output, async (event) => {
      await ffmpegOutputAction(event);
    });

    const unsubscribeFFmpegProgress = Events.On(FFmpegEventType.Progress, (event) => {
      ffmpegProgressAction(event);
    });

    const unsubscribeFFmpegCompleted = Events.On(FFmpegEventType.Completed, (_) => {
      progress = 100.0
    });

    loadFFmpegPath()

    window.addEventListener('wheel', (event) => { _disableCtrlWheel(event) }, { passive: false });
    window.addEventListener('keydown', (event) => { _disableZoomKey(event) }, { passive: false });

    return () => {
      unsubscribeDrop();
      unsubscribeFFmpegOutput();
      unsubscribeFFmpegProgress();
      unsubscribeFFmpegCompleted();
    };
  });

  /**
   * 處理「影片檔案拖放完成」事件的回調函式 + 
   * - 從事件中解出 string[] 類型的檔案路徑陣列
   * - 若陣列非空，則將第一個路徑設為 inputPath 並更新日誌
   * - 若無有效路徑，則顯示錯誤訊息
   *
   * @param event - 事件物件（通常來自 Events 系統）
   */
  async function videoFileDroppedAction(event: unknown): Promise<void> {

    const files = _getEventData<string[]>(event);

    if (!Array.isArray(files) || files.length === 0) {
      logText = "未能讀取到拖放的檔案路徑";
      videoDuration = 0;
      videoDurationText = "";
      return;
    }

    inputPath = files[0];
    logText = `已成功選擇媒體：${inputPath}`;
    saveFFmpegPath(ffmpegPath)

    try {
      videoDuration = await GetVideoDuration(ffmpegPath, inputPath);
      videoDurationText = _formatDuration(videoDuration);
      
      isVideo = (await DetectMediaType(ffmpegPath, inputPath) == "video")
      if (!isVideo) { videoCodec = VideoCodecs.Copy }

      logText += `\n媒體長度：${videoDurationText}`;
    } catch (error) {
      videoDuration = 0;
      videoDurationText = "";
      logText += `\n${String(error)}`;
    }
  }

  /**
   * 處理 FFmpeg 即時輸出行的事件回調
   *
   * - 從事件中解出 FFmpegOutput 物件
   * - 若 output 存在且 output.line 為字串，則追加到日誌並捲動到底部
   * - 否則忽略該事件
   *
   * @param event - 來自 Events 系統的事件物件，負載為 FFmpegOutput
   */
  async function ffmpegOutputAction(event: unknown): Promise<void> {
    const output = _getEventData<FFmpegOutput>(event);

    if (!output || typeof output.line !== "string") { return; }

    _appendLog(output.line);
    await _scrollLogToBottom();
  }

  /**
   * 處理後端 FFmpeg 發送的轉換進度事件
   *
   * 後端會透過 `ffmpeg:progress` 事件傳送 FFmpegProgress：
   * - currentSeconds：目前已處理的影片秒數
   * - totalSeconds：影片總長度（秒）
   * - percent：目前轉換百分比
   * - 這裡只使用 percent 更新 UI 的進度條，並將數值限制為 0～100，避免因 FFmpeg 最後 timestamp、浮點數誤差或異常資料，造成 progress element 收到負數或超過 100 的數值
   */
  function ffmpegProgressAction(event: unknown) {
    const data = _getEventData<FFmpegProgress>(event);
    progress = Math.min(100, Math.max(0, data.percent));
  }

  /**
   * 轉換按鈕的點擊處理函式：
   * - 若正在轉檔，則呼叫取消轉檔
   * - 否則開始新的轉檔流程
   */
  async function handleConvertButton() {
    if (converting) {
      await cancelConversion();
      return;
    }

    await startConversion();
  }

  /**
   * 開始執行影片轉檔流程：
   * 1. 檢查輸入是否正確
   * 2. 設定轉換中狀態並顯示參數預覽日誌
   * 3. 呼叫 StartConversion 進行實際轉檔
   * 4. 根據成功/失敗更新日誌並捲動到底部
   * 5. 最後重置狀態
   */
  async function startConversion() {
    const error = _checkInputError(inputPath, ffmpegPath);

    if (error) {
      logText = error;
      return;
    }

    converting = true;
    logText = _combineLogText();

    try {
      const result = await StartConversion({
        inputPath,
        ffmpegPath,
        container,
        videoCodec,
        startTime,
        endTime,
        width,
        height,
        useStart: enableStart,
        useEnd: enableEnd,
        useSize: enableSize,
      });

      logText += `${result.message}`;
      await _scrollLogToBottom();
    } catch (error) {
      Print(String(error))
      dialog("error", "錯誤", String(error))
      logText += `\n\n----- 轉換失敗 -----\n${String(error)}`;
      await _scrollLogToBottom();
    } finally {
      converting = false;
      cancelling = false;
    }
  }

  /**
   * 顯示原生系統對話框
   *
   *  - 依照傳入的 type 決定要顯示一般提示、錯誤或警告對話框；title 會顯示在對話框標題，message 則是對話框內容
   *
   * @param type 對話框類型：info、error 或 warning
   * @param title 對話框標題
   * @param message 對話框顯示的訊息內容
   */
  async function dialog(type: DialogType, title: string, message: string) {

    switch (type) {
        case "info": await Dialogs.Info({ Title: title, Message: message }); break;
        case "error": await Dialogs.Error({ Title: title, Message: message }); break;
        case "warning": await Dialogs.Warning({ Title: title, Message: message }); break;
        default: await Dialogs.Info({ Title: title, Message: message });
      }
  }

  /**
   * 請求取消目前的轉檔作業
   * 1. 若已在取消中，直接返回（防重複）
   * 2. 設定 cancelling 旗標
   * 3. 呼叫 CancelConversion 請求後端/本地安全停止 FFmpeg
   * 4. 依回傳結果更新日誌或重置旗標
   */
  async function cancelConversion() {
    if (cancelling) {
      return;
    }

    cancelling = true;

    try {
      const accepted = await CancelConversion();

      if (accepted) {
        await _scrollLogToBottom();
      } else {
        cancelling = false;
      }
    } catch (error) {
      cancelling = false;
      logText += `\n取消失敗：${String(error)}\n`;
      await _scrollLogToBottom();
    }
  }

  /**
   * 檢查轉檔前的必要輸入是否正確
   * @param inputPath - 輸入影片路徑
   * @param ffmpegPath - FFmpeg 執行檔路徑
   * @returns 若有錯誤則回傳錯誤訊息，否則回傳 null
   */
  function _checkInputError(inputPath: string, ffmpegPath: string): string | null {
    if (!inputPath) {
      return "錯誤：請先選擇輸入影片";
    }

    if (!ffmpegPath.trim()) {
      return "錯誤：請輸入 FFmpeg 執行檔路徑";
    }

    return null;
  }

  /**
   * 組合轉檔前的參數預覽日誌
   * @returns 多行字串形式的日誌內容
   */
  function _combineLogText(): string {
    const log = [
      "正在呼叫 FFmpeg...",
      `輸入：${inputPath}`,
      `格式：.${container}`,
      `編碼器：${videoCodec}`,
      enableStart ? `開始：${startTime}` : "不限制開始時間",
      enableEnd ? `結束：${endTime}` : "不限制結束時間",
      enableSize ? `尺寸：${width} × ${height}` : "維持原始尺寸",
    ].join("\n");

    return log;
  }

  /**
   * 將一筆 FFmpeg 輸出追加到前端 log
   *
   * - FFmpeg 的即時進度列通常以 `\r` 結尾，而非 `\n`
   * - `\r` 在終端機中代表游標回到目前行首，因此下一筆狀態會覆寫同一行；這裡模擬相同行為，將 logText 的最後一行替換為最新進度資訊，避免每次 progress 更新都新增一行文字
   * - 一般以 `\n` 結尾的 FFmpeg metadata、warning、error 或一般訊息，則視為獨立 log 行，直接追加到 logText
   */
  function _appendLog(message: string) {
    if (message.endsWith("\r")) {
      const currentLine = message.slice(0, -1);
      const lines = logText.split("\n");

      lines[lines.length - 1] = currentLine;
      logText = lines.join("\n");
      return;
    }

    const normalized = message.replace(/\r\n/g, "\n");
    logText += normalized.endsWith("\n") ? normalized : `${normalized}\n`;
  }

  /**
   * 從事件物件中安全取出泛型資料 T
   * - 若 event 是物件且有 data 欄位，則回傳 event.data
   * - 否則直接把 event 當作 T 回傳
   *
   * @template T - 期望取得的資料型別
   * @param event - 事件物件或任意值
   * @returns 解包後的資料，型別為 T
   */
  function _getEventData<T>(event: unknown): T {
    if (event !== null && typeof event === "object" && "data" in event) {
      return (event as { data: T }).data;
    }

    return event as T;
  }

  /**
   * 將秒數格式化為固定的 HH:MM:SS 時間字串。
   *
   * 小數秒會向下取整；負數會先限制為 0。
   *
   * 範例：
   * - 0 → "00:00:00"
   * - 51.54 → "00:00:51"
   * - 764 → "00:12:44"
   * - 3_661 → "01:01:01"
   */
  function _formatDuration(totalSeconds: number): string {
    const total = Math.max(0, Math.floor(totalSeconds));
    const hours = Math.floor(total / 3600);
    const minutes = Math.floor((total % 3600) / 60);
    const seconds = total % 60;

    return [
      hours.toString().padStart(2, "0"),
      minutes.toString().padStart(2, "0"),
      seconds.toString().padStart(2, "0"),
    ].join(":");
  }

  /**
   * 攔截鍵盤事件，禁止使用者用 Ctrl/Cmd + +/-/0 進行頁面縮放。
   * 適用於 Wails 前端（或其他 webview），搭配 CSS touch-action: none 一起使用，
   * 可以大幅壓制桌面端常見的縮放快捷鍵。
   */
  function _disableZoomKey(event: KeyboardEvent) {

    const isZoomControl = event.ctrlKey || event.metaKey;
    const isZoomKey = event.key === '+' || event.key === '-' || event.key === '0' || event.key === '=';

    if (isZoomControl && isZoomKey) {
      event.preventDefault();
    }
  }

  /**
   * 攔截滾輪事件，禁止使用者用 Ctrl + 滾輪進行頁面縮放。
   * 適用於桌面端瀏覽器 / webview（包含 Wails），搭配 CSS touch-action: none
   * 與鍵盤縮放攔截一起使用，可大幅壓制常見的縮放操作。
   */
  function _disableCtrlWheel(event: WheelEvent) {
    if (event.ctrlKey) { event.preventDefault(); }
  }

  /**
   * 將日誌區域捲動到最底部
   *  - 先等待 DOM 更新（tick），再設定 scrollTop
   */
  async function _scrollLogToBottom(): Promise<void> {
    await tick();
    if (!logElement) { return; }

    logElement.scrollTop = logElement.scrollHeight;
  }
</script>

<svelte:head>
  <title>影片格式轉換器</title>
</svelte:head>

<main class="window">
  <section class="drop-zone" data-file-drop-target>
    <div class="form-area">
      <FFmpegPathInput bind:value={ffmpegPath} disabled={converting} />
      <div class="form-row file-row">
        <FilePathInput value={inputPath} />
        <OutputFormatSelect bind:value={container} disabled={converting} />
        <VideoCodecSelect bind:value={videoCodec} disabled={converting || !isVideo} />
        <ConvertButton {videoDuration} {converting} {cancelling} hasInput={Boolean(inputPath)} onclick={handleConvertButton} />
      </div>

      <div class="options-grid">
        <TimeOptionRow label="開始" ariaLabel="開始時間" placeholder="00:00:00" bind:enabled={enableStart} bind:time={startTime} {converting} />
        <TimeOptionRow label="結束" ariaLabel="結束時間" placeholder="23:59:59" bind:enabled={enableEnd} bind:time={endTime} {converting} />
        <SizeOptionRow bind:enabled={enableSize} bind:width bind:height {videoCodec} {converting} />
      </div>

      <div class="progress-area" class:visible={converting || progress > -1} aria-live="polite">
        <progress value={progress} max="100" aria-label="轉換進度"></progress>
        <span class="progress-value">{progress.toFixed(0)}%</span>
      </div>

      <pre class="log-panel" bind:this={logElement}>{logText}</pre>
    </div>
  </section>
</main>
