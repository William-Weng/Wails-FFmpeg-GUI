# Wails FFmpeg GUI

![Go](https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white)
![Node.js](https://img.shields.io/badge/Node.js-20.19.2-339933?logo=nodedotjs&logoColor=white)
![Wails](https://img.shields.io/badge/Wails-v3.0.0--beta.20-DF0000?logo=wails&logoColor=white)
![LICENSE](https://img.shields.io/github/license/William-Weng/Wails-FFmpeg-GUI?style=flat&label=LICENSE&color=yellow)
![Tag](https://img.shields.io/github/v/tag/William-Weng/Wails-FFmpeg-GUI?style=flat&label=Tag)
![Stars](https://img.shields.io/github/stars/William-Weng/Wails-FFmpeg-GUI?style=flat&label=Stars)

一個使用 Wails 3、Go、Svelte 5 與 FFmpeg 製作的跨平台影片格式轉換工具。

目前主要以 macOS 開發與測試，提供 FFmpeg 指令預覽、複製與影片轉換功能，方便在圖形化介面與 Terminal 指令之間切換。

## 功能特色

- 透過圖形化介面設定輸入與輸出影片。
- 支援 FFmpeg command 預覽。
- 可直接複製產生的 FFmpeg 指令到 Terminal 使用。
- 支援 `copy` 模式，避免不必要的重新編碼。
- 可設定影片裁切時間與輸出尺寸。
- Go backend 負責執行 FFmpeg 外部程序。
- Svelte 5 frontend 負責表單狀態與操作介面。
- 使用 Less 管理前端樣式。

## 技術架構

```text
Svelte 5 + Less
        │
        │ Wails bindings
        ▼
Go service
        │
        │ os/exec
        ▼
FFmpeg
```

## 作業環境

| 項目 | 版本 |
| --- | --- |
| [macOS](https://www.apple.com/tw/os/macos/) | Sequoia 15.7 |
| [Xcode](https://developer.apple.com/xcode/) | 26.3 |
| [Go](https://go.dev/) | 1.27.1 |
| [Node.js](https://nodejs.org) | 20.19.2 |
| [NPM](https://www.npmjs.com/) | 11.5.2 |
| [Wails](https://v3.wails.io) | v3.0.0-beta.20 |
| [Svelte](https://svelte.dev/) | 5 |
| [Less](https://lesscss.org/) | 依 `package.json` 為準 |

## 系統需求

執行本專案前，請先安裝：

- macOS 與 Xcode Command Line Tools
- Go
- Node.js 與 NPM
- Wails 3 CLI
- FFmpeg

macOS Apple Silicon 可使用 Homebrew 安裝 FFmpeg：

```bash
brew install ffmpeg
which ffmpeg
```

若程式使用固定路徑，Apple Silicon 通常是：

```text
/opt/homebrew/bin/ffmpeg
```

Intel Mac 常見路徑則是：

```text
/usr/local/bin/ffmpeg
```

建議在應用程式中提供 FFmpeg 路徑設定，而不要永遠寫死單一路徑。

## 建立專案

列出可用的 Wails 3 frontend templates：

```bash
wails3 init -l
```

建立 Vanilla 專案：

```bash
wails3 init -n ffmpeg-converter -t vanilla
```

建立 Svelte 專案：

```bash
wails3 init -n ffmpeg-converter -t svelte
```

## 開發

安裝 frontend dependencies：

```bash
cd frontend
npm install
npm install -D less
cd ..
```

啟動開發模式：

```bash
wails3 dev
```

若需要查詢或終止佔用開發 port 的程序：

```bash
lsof -i:9245
kill <PID>
```

請將 `<PID>` 替換成 `lsof` 查到的實際程序 ID，不要直接固定使用舊的 PID。

## FFmpeg 指令

程式會依照表單設定組合 FFmpeg arguments，例如：

```bash
ffmpeg -hide_banner -y -i input.mp4 -c copy output.ts
```

產生的 command 可以直接複製到 Terminal，適合用於除錯、重複執行或進一步調整參數。

### `copy` 模式注意事項

`-c copy` 只複製原始音訊與影片串流，不會重新編碼，因此速度通常較快且不會降低畫質。

但是，`-c copy` 不能和縮放等影片濾鏡同時使用。例如以下組合需要重新編碼：

```bash
ffmpeg -i input.mp4 -vf scale=1920:1080 -c:v libx264 output.mp4
```

如果設定了輸出尺寸，請使用支援重新編碼的 codec，而不是 `copy`。

## 建置與封裝

建立 Windows amd64 binary：

```bash
wails3 build GOOS=windows GOARCH=amd64
```

封裝 macOS Intel 版本：

```bash
wails3 package GOOS=darwin GOARCH=amd64
```

封裝 macOS Apple Silicon 版本：

```bash
wails3 package GOOS=darwin GOARCH=arm64
```

執行格式化：

```bash
gofmt -w main.go ffmpeg_service.go
gofmt -w *.go
```

應用程式 icon：

```text
build/appicon.png
```

## Wails Tasks

列出可用 tasks：

```bash
wails3 task --list
```

設定 Docker 環境：

```bash
wails3 task setup:docker
```

建立 Windows amd64：

```bash
wails3 task windows:build ARCH=amd64
```

執行封裝 task：

```bash
wails3 task package
```

## 專案結構

```text
.
├── build/                 # 建置資源與應用程式 icon
├── frontend/              # Svelte 5 frontend
├── ffmpeg_service.go      # FFmpeg command 組合與執行
├── main.go                # Wails application entry point
├── go.mod                 # Go module 設定
├── go.sum                 # Go dependencies checksum
└── wails.json             # Wails 專案設定
```

實際檔案名稱若與上述不同，請以專案目前結構為準。

## 開發筆記

- 先確認產生的 FFmpeg command 可以在 Terminal 正常執行，再進行 GUI 內的實際轉換。
- 不要把使用者輸入直接拼成 shell command 後交給 shell 執行；Go backend 應使用 `exec.CommandContext` 分開傳入 executable 與 arguments。
- 若要讓 command 可複製，應對每個 argument 做 shell quoting，尤其要處理空白、中文路徑與單引號。
- FFmpeg 的輸出通常會寫到 stderr，若要顯示即時進度，應讀取 stderr pipe 並透過 Wails event 傳到 frontend。
- 不要固定寫死 `kill 79647` 這類舊 PID；每次執行前應以 `lsof` 查詢最新 PID。

