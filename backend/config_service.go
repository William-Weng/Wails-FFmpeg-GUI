package backend

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	util "ffmpeg-gui/backend/utility"
)

// ConfigService 負責管理應用程式設定的讀取、修改與寫入
//
// ~/Library/Application Support/VideoConverter/config.json
type ConfigService struct {
	configPath string         //configPath 是設定檔在作業系統中的完整位置
	config     util.AppConfig //是目前已載入記憶體中的設定內容
}

// 建立設定服務，並在啟動時讀取既有設定
//
// 例如：
// macOS:   ~/Library/Application Support/VideoConverter
// Windows: %AppData%\VideoConverter
// Linux:   ~/.config/VideoConverter
func NewConfigService(appName string) (*ConfigService, error) {

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	appDir := filepath.Join(configDir, appName)

	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return nil, err
	}

	service := &ConfigService{
		configPath: filepath.Join(appDir, "config.json"),
	}

	if err := service.load(); err != nil {
		return nil, err
	}

	return service, nil
}

// 取得 FFmpeg 路徑
func (service *ConfigService) GetFFmpegPath() string {
	return service.config.FFmpegPath
}

// 儲存 FFmpeg 路徑
func (service *ConfigService) SetFFmpegPath(path string) error {
	service.config.FFmpegPath = path
	return service.save()
}

// 從 config.json 載入設定
func (service *ConfigService) load() error {

	data, err := os.ReadFile(service.configPath)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	return json.Unmarshal(data, &service.config)
}

// 將設定寫入 config.json
//
// - 先寫入暫存檔，再替換正式檔案，避免程式中斷造成設定檔損壞
func (service *ConfigService) save() error {

	data, err := json.MarshalIndent(service.config, "", "  ")
	if err != nil {
		return err
	}

	tempPath := service.configPath + ".tmp"

	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tempPath, service.configPath)
}
