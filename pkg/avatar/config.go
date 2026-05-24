package avatar

import (
	"encoding/json"
	"os"
)

// Config – настройки проекта, которые можно переопределить при использовании как библиотеки.
type Config struct {
	ImageOutWidth    int
	ImageOutHeight   int
	ImageOutFontSize int
	ImageOutFontPath string

	ErrorBgColor   string
	ErrorTextColor string

	SuccessBgColor   string
	SuccessTextColor string

	AvatarWidth     int
	AvatarHeight    int
	AvatarFontSize  int
	AvatarTextColor string

	DarkColor  string
	LightColor string

	DirPermission0755  os.FileMode
	FilePermission0644 os.FileMode

	SaveAvatarToPath      string
	MaxTextTruncateLength int
}

// NewDefaultConfig возвращает конфигурацию со значениями по умолчанию.
func NewDefaultConfig() *Config {
	return &Config{
		ImageOutWidth:    400,
		ImageOutHeight:   300,
		ImageOutFontSize: 20,
		ImageOutFontPath: "fonts/ShareTechMono.ttf",

		ErrorBgColor:   "#ffffff",
		ErrorTextColor: "#e21613",

		SuccessBgColor:   "#ffffff",
		SuccessTextColor: "#000000",

		AvatarWidth:     300,
		AvatarHeight:    300,
		AvatarFontSize:  45,
		AvatarTextColor: "#ffffff",

		DarkColor:  "#000000",
		LightColor: "#ffffff",

		DirPermission0755:  0755,
		FilePermission0644: 0644,

		SaveAvatarToPath:      "uploads",
		MaxTextTruncateLength: 1,
	}
}

var AppConfig *Config

// LoadConfig загружает конфигурацию, если передан путь, иначе использует значения по умолчанию
func LoadConfig(path string) error {
	if path == "" {
		AppConfig = NewDefaultConfig()
		return nil
	}

	// Загрузка из JSON файла
	file, err := os.Open(path)
	if err != nil {
		AppConfig = NewDefaultConfig()
		return nil // не фатально, используем дефолт
	}
	defer file.Close()

	var jsonConfig map[string]interface{}
	if err := json.NewDecoder(file).Decode(&jsonConfig); err != nil {
		AppConfig = NewDefaultConfig()
		return nil
	}

	AppConfig = NewDefaultConfig()

	if fontPath, ok := jsonConfig["font_path"].(string); ok && fontPath != "" {
		AppConfig.ImageOutFontPath = fontPath
	}
	if fontSize, ok := jsonConfig["default_font_size"].(float64); ok && fontSize > 0 {
		AppConfig.AvatarFontSize = int(fontSize)
	}
	if imgSize, ok := jsonConfig["image_size"].(float64); ok && imgSize > 0 {
		AppConfig.AvatarWidth = int(imgSize)
		AppConfig.AvatarHeight = int(imgSize)
	}
	if maxLen, ok := jsonConfig["max_email_prefix"].(float64); ok && maxLen > 0 {
		AppConfig.MaxTextTruncateLength = int(maxLen)
	}
	if colorsPath, ok := jsonConfig["colors_json_path"].(string); ok && colorsPath != "" {
		// путь к файлу цветов, может быть использован в LoadColors
	}

	return nil
}
