<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { EventType } from "./constants/events";

  import FFmpegPathInput from "./components/FfmpegPathInput.svelte";
  import FilePathInput from "./components/FilePathInput.svelte";
  import OutputFormatSelect from "./components/OutputFormatSelect.svelte";
  import VideoCodecSelect from "./components/VideoCodecSelect.svelte";
  import ConvertButton from "./components/ConvertButton.svelte";
  import TimeOptionRow from "./components/TimeOptionRow.svelte";
  import SizeOptionRow from "./components/SizeOptionRow.svelte";

  import {
    StartConversion,
    CancelConversion,
  } from "../bindings/changeme/ffmpegservice";

  type FFmpegOutput = {
    line: string;
  };

  let ffmpegPath = "/opt/homebrew/bin/ffmpeg";
  let inputPath = "";
  let container: "mp4" | "mkv" | "webm" | "ts" = "mp4";
  let videoCodec: "copy" | "h264" | "h265" = "copy";

  let enableStart = false;
  let enableEnd = false;
  let enableSize = false;
  let converting = false;
  let cancelling = false;

  let startTime = "00:00:00";
  let endTime = "00:01:00";
  let width = 1920;
  let height = 1080;

  let logText = "請拖放影片檔案到上方區域";
  let logElement: HTMLElement | null = null;
  
  /**
   * 元件掛載時註冊事件監聽器：
   * 1. "video-file-dropped"：處理影片檔案拖放完成
   * 2. "ffmpeg:output"：處理 FFmpeg 即時輸出行
   *
   * 回傳的清理函式會在元件卸載時取消所有訂閱
   */
  onMount(() => {
    const unsubscribeDrop = Events.On(EventType.VideoFileDropped, (event) => {
      videoFileDroppedAction(event);
    });

    const unsubscribeFFmpegOutput = Events.On(
      EventType.FFmpegOutput,
      async (event) => {
        await ffmpegOutputAction(event);
      },
    );

    return () => {
      unsubscribeDrop();
      unsubscribeFFmpegOutput();
    };
  });

  /**
   * 處理「影片檔案拖放完成」事件的回調函式
   * - 從事件中解出 string[] 類型的檔案路徑陣列
   * - 若陣列非空，則將第一個路徑設為 inputPath 並更新日誌
   * - 若無有效路徑，則顯示錯誤訊息
   *
   * @param event - 事件物件（通常來自 Events 系統）
   */
  function videoFileDroppedAction(event: unknown): void {
    const files = getEventData<string[]>(event);

    if (Array.isArray(files) && files.length > 0) {
      inputPath = files[0];
      logText = `已成功選擇影片：${inputPath}`;
      console.log("拖放的檔案路徑：", inputPath);
      return;
    }

    logText = "未能讀取到拖放的檔案路徑";
  }

  /**
   * 處理 FFmpeg 即時輸出行的事件回調
   * - 從事件中解出 FFmpegOutput 物件
   * - 若 output 存在且 output.line 為字串，則追加到日誌並捲動到底部
   * - 否則忽略該事件
   *
   * @param event - 來自 Events 系統的事件物件，負載為 FFmpegOutput
   */
  async function ffmpegOutputAction(event: unknown): Promise<void> {
    const output = getEventData<FFmpegOutput>(event);

    if (!output || typeof output.line !== "string") {
      return;
    }

    logText += `${output.line}\n`;
    await _scrollLogToBottom();
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
  function getEventData<T>(event: unknown): T {
    if (event !== null && typeof event === "object" && "data" in event) {
      return (event as { data: T }).data;
    }

    return event as T;
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

      logText += `\n----- 轉換完成 -----\n${result.message}`;
      await _scrollLogToBottom();
    } catch (error) {
      logText += `\n----- 轉換失敗 -----\n${String(error)}`;
      await _scrollLogToBottom();
    } finally {
      converting = false;
      cancelling = false;
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
        logText += "\n----- 正在安全停止 FFmpeg -----\n";
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
  function _checkInputError(
    inputPath: string,
    ffmpegPath: string,
  ): string | null {
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
   * 將日誌區域捲動到最底部
   *  - 先等待 DOM 更新（tick），再設定 scrollTop
   */
  async function _scrollLogToBottom(): Promise<void> {
    await tick();

    if (!logElement) {
      return;
    }

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
        <VideoCodecSelect bind:value={videoCodec} disabled={converting} />
        <ConvertButton
          {converting}
          {cancelling}
          hasInput={Boolean(inputPath)}
          onclick={handleConvertButton}
        />
      </div>

      <div class="options-grid">
        <TimeOptionRow
          label="開始"
          ariaLabel="開始時間"
          placeholder="00:00:00"
          bind:enabled={enableStart}
          bind:time={startTime}
          {converting}
        />

        <TimeOptionRow
          label="結束"
          ariaLabel="結束時間"
          placeholder="23:59:59"
          bind:enabled={enableEnd}
          bind:time={endTime}
          {converting}
        />

        <SizeOptionRow
          bind:enabled={enableSize}
          bind:width
          bind:height
          {videoCodec}
          {converting}
        />
      </div>

      <pre class="log-panel" bind:this={logElement}>{logText}</pre>
    </div>
  </section>
</main>
