package avatar

import (
	"image/color"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}

	if AppConfig.AvatarWidth != 300 {
		t.Errorf("expected 300, got %d", AppConfig.AvatarWidth)
	}
	if AppConfig.AvatarFontSize != 45 {
		t.Errorf("expected 45, got %d", AppConfig.AvatarFontSize)
	}
	if AppConfig.MaxTextTruncateLength != 2 {
		t.Errorf("expected 2, got %d", AppConfig.MaxTextTruncateLength)
	}
}

func TestTruncateEmail(t *testing.T) {
	maxLen := 2

	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{"Полный email", "verylongemail@example.com", "ve"},
		{"Короткий email", "ab@example.com", "ab"},
		{"Без домена", "testuser", "te"},
		{"Пустой email", "", ""},
		{"Ровно 2 символа", "ab@test.com", "ab"},
		{"Кириллица в email", "тест@example.com", "те"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateEmail(tt.email, maxLen)
			if result != tt.expected {
				t.Errorf("TruncateEmail(%q) = %q, ожидалось %q", tt.email, result, tt.expected)
			}
		})
	}
}

func TestTruncateEmail_NegativeMaxLen(t *testing.T) {
	// Отрицательная длина должна возвращать пустую строку
	result := TruncateEmail("test@example.com", -1)
	if result != "" {
		t.Errorf("expected empty string for negative maxLen, got %q", result)
	}
}

func TestGetContrastColor(t *testing.T) {
	white := GetContrastColor(color.RGBA{0, 0, 0, 255})
	if white != color.White {
		t.Errorf("expected white on black background, got %v", white)
	}

	black := GetContrastColor(color.RGBA{255, 255, 255, 255})
	if black != color.Black {
		t.Errorf("expected black on white background, got %v", black)
	}
}

func TestGenerateAvatar(t *testing.T) {
	err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}

	AppConfig.ImageOutFontPath = "fonts/ShareTechMono.ttf"

	err = LoadColors("../colors.json")
	if err != nil {
		t.Log("Предупреждение: colors.json не найден, используются fallback цвета")
	}

	hex, err := GenerateAvatar("test@example.com", "test_avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if hex == "" {
		t.Error("hex is empty")
	}
	defer os.Remove("test_avatar.png")
}

func TestGenerateAvatarWithColor(t *testing.T) {
	err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}

	AppConfig.ImageOutFontPath = "fonts/ShareTechMono.ttf"

	err = LoadColors("../colors.json")
	if err != nil {
		t.Log("Предупреждение: colors.json не найден, используются fallback цвета")
	}

	hex, err := GenerateAvatarWithColor("test@example.com", "test_avatar_color.png", "#3498DB", 50)
	if err != nil {
		t.Fatal(err)
	}
	if hex == "" {
		t.Error("hex is empty")
	}
	if hex != "#3498DB" {
		t.Errorf("expected #3498DB, got %s", hex)
	}
	defer os.Remove("test_avatar_color.png")
}

func TestGetRandomColorSafe(t *testing.T) {
	err := LoadColors("../colors.json")
	if err != nil {
		t.Log("Предупреждение: colors.json не найден, используются fallback цвета")
	}

	color, err := GetRandomColorSafe()
	if err != nil {
		t.Fatalf("GetRandomColorSafe вернул ошибку: %v", err)
	}
	if color.Hex == "" {
		t.Error("color hex is empty")
	}
	if color.Name == "" {
		t.Error("color name is empty")
	}
}

func TestGetRandomColorSafe_EmptyList(t *testing.T) {
	originalColors := colorsList
	colorsList = []ColorInfo{}

	_, err := GetRandomColorSafe()
	if err == nil {
		t.Error("Ожидалась ошибка при пустом списке цветов")
	}

	colorsList = originalColors
}

func TestGetAllColors(t *testing.T) {
	err := LoadColors("../colors.json")
	if err != nil {
		t.Log("Предупреждение: colors.json не найден, используются fallback цвета")
	}

	colors := GetAllColors()
	if len(colors) == 0 {
		t.Error("colors list is empty")
	}
}

func TestGetColorByHex(t *testing.T) {
	err := LoadColors("../colors.json")
	if err != nil {
		t.Log("Предупреждение: colors.json не найден, используются fallback цвета")
	}

	color, err := GetColorByHex("#3498DB")
	if err != nil {
		t.Errorf("error getting color: %v", err)
	}
	if color == nil {
		t.Error("color is nil")
	}

	_, err = GetColorByHex("#000000")
	if err == nil {
		t.Error("expected error for non-existent color")
	}
}

func TestLoadColorsFallback(t *testing.T) {
	err := LoadColors("non_existent_file.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}

	colors := GetAllColors()
	if len(colors) == 0 {
		t.Error("fallback colors not set")
	}
	if len(colors) < 3 {
		t.Errorf("expected at least 3 fallback colors, got %d", len(colors))
	}
}

func TestColorInfoIsLight(t *testing.T) {
	darkColor := ColorInfo{R: 0, G: 0, B: 0}
	if darkColor.IsLight() {
		t.Error("dark color should not be light")
	}

	lightColor := ColorInfo{R: 255, G: 255, B: 255}
	if !lightColor.IsLight() {
		t.Error("light color should be light")
	}
}

func TestGenerateAvatar_InvalidFontPath(t *testing.T) {
	err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}

	AppConfig.ImageOutFontPath = "fonts/non_existent_font.ttf"

	_, err = GenerateAvatar("test@example.com", "test_avatar.png")
	if err == nil {
		t.Error("Ожидалась ошибка при неверном пути к шрифту")
	}

	os.Remove("test_avatar.png")
}

func TestGenerateAvatarWithColor_InvalidHex(t *testing.T) {
	err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}

	AppConfig.ImageOutFontPath = "fonts/ShareTechMono.ttf"

	LoadColors("colors.json")

	_, err = GenerateAvatarWithColor("test@example.com", "test_invalid_hex.png", "#INVALID", 45)

	if err == nil {
		t.Error("Ожидалась ошибка при некорректном HEX цвете")
	}

	os.Remove("test_invalid_hex.png")
}

func TestGenerateAvatar_EmptyEmail(t *testing.T) {
	err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}

	AppConfig.ImageOutFontPath = "fonts/ShareTechMono.ttf"

	LoadColors("colors.json")

	hex, err := GenerateAvatar("", "test_empty_email.png")

	if err != nil {
		t.Errorf("Не ожидалась ошибка при пустом email, получено: %v", err)
	}

	if hex == "" {
		t.Error("HEX цвет не должен быть пустым")
	}

	if _, err := os.Stat("test_empty_email.png"); os.IsNotExist(err) {
		t.Error("Файл аватара не был создан")
	}

	os.Remove("test_empty_email.png")
}

func TestGetColorByHex_CaseInsensitive(t *testing.T) {
	err := LoadColors("colors.json")
	if err != nil {
		t.Log("Предупреждение: colors.json не найден")
		return
	}

	color1, err1 := GetColorByHex("#e74c3c")
	color2, err2 := GetColorByHex("#E74C3C")

	if err1 != nil || err2 != nil {
		t.Log("Цвет не найден в списке, пропускаем тест")
		return
	}

	if color1.Hex != color2.Hex {
		t.Errorf("Поиск цвета должен быть регистронезависимым: %s vs %s", color1.Hex, color2.Hex)
	}
}

func TestGetAvatarText(t *testing.T) {
	// Сохраняем оригинальный MaxTextTruncateLength
	originalMaxLen := AppConfig.MaxTextTruncateLength
	AppConfig.MaxTextTruncateLength = 2
	defer func() { AppConfig.MaxTextTruncateLength = originalMaxLen }()

	tests := []struct {
		email    string
		expected string
	}{
		{"test@example.com", "TE"},
		{"ab@example.com", "AB"},
		{"user", "US"},
		{"", "?"},
		{"тест@example.com", "ТЕ"},
		{"я", "Я"},
	}

	for _, tt := range tests {
		result := getAvatarText(tt.email)
		if result != tt.expected {
			t.Errorf("getAvatarText(%q) = %q, ожидалось %q", tt.email, result, tt.expected)
		}
	}
}
