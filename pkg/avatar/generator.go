package avatar

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"sync"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
)

type ColorInfo struct {
	Name string `json:"name"`
	Hex  string `json:"hex"`
	R    int    `json:"r"`
	G    int    `json:"g"`
	B    int    `json:"b"`
}

var colorsList []ColorInfo
var (
	cachedFont     *truetype.Font
	cachedFontPath string
	fontMutex      sync.RWMutex
)

func getFont(fontPath string) (*truetype.Font, error) {
	fontMutex.RLock()
	if cachedFont != nil && cachedFontPath == fontPath {
		fontMutex.RUnlock()
		return cachedFont, nil
	}
	fontMutex.RUnlock()

	fontMutex.Lock()
	defer fontMutex.Unlock()

	if cachedFont != nil && cachedFontPath == fontPath {
		return cachedFont, nil
	}

	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки шрифта: %w", err)
	}

	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга шрифта: %w", err)
	}

	cachedFont = f
	cachedFontPath = fontPath
	return cachedFont, nil
}

func LoadColors(path string) error {
	file, err := os.Open(path)
	if err != nil {
		colorsList = getFallbackColors()
		return fmt.Errorf("не удалось загрузить файл цветов (%s), используются стандартные цвета: %w", path, err)
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&colorsList)
	if err != nil {
		colorsList = getFallbackColors()
		return fmt.Errorf("ошибка парсинга файла цветов (%s), используются стандартные цвета: %w", path, err)
	}

	if len(colorsList) == 0 {
		colorsList = getFallbackColors()
		return fmt.Errorf("файл цветов (%s) пуст, используются стандартные цвета", path)
	}

	return nil
}

func getFallbackColors() []ColorInfo {
	return []ColorInfo{
		{Name: "Красный", Hex: "#E74C3C", R: 231, G: 76, B: 60},
		{Name: "Синий", Hex: "#3498DB", R: 52, G: 152, B: 219},
		{Name: "Зелёный", Hex: "#2ECC71", R: 46, G: 204, B: 113},
		{Name: "Оранжевый", Hex: "#F39C12", R: 243, G: 156, B: 18},
		{Name: "Фиолетовый", Hex: "#9B59B6", R: 155, G: 89, B: 182},
		{Name: "Тёмно-синий", Hex: "#34495E", R: 52, G: 73, B: 94},
	}
}

func secureRandomInt(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("n должно быть положительным")
	}
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return 0, err
	}
	return int(binary.LittleEndian.Uint64(b[:]) % uint64(n)), nil
}

// GetRandomColorSafe возвращает случайный цвет без паники
func GetRandomColorSafe() (ColorInfo, error) {
	if len(colorsList) == 0 {
		return ColorInfo{}, fmt.Errorf("список цветов пуст")
	}
	idx, err := secureRandomInt(len(colorsList))
	if err != nil {
		return colorsList[0], nil
	}
	return colorsList[idx], nil
}

func GetAllColors() []ColorInfo {
	return colorsList
}

func GetColorByHex(hexColor string) (*ColorInfo, error) {
	for _, color := range colorsList {
		if strings.EqualFold(color.Hex, hexColor) {
			return &color, nil
		}
	}
	return nil, fmt.Errorf("цвет с HEX %s не найден", hexColor)
}

func GetContrastColor(bgColor color.RGBA) color.Color {
	brightness := 0.299*float64(bgColor.R) + 0.587*float64(bgColor.G) + 0.114*float64(bgColor.B)
	if brightness > 128 {
		return color.Black
	}
	return color.White
}

// TruncateEmail обрезает email до указанной длины (поддержка кириллицы)
func TruncateEmail(email string, maxLen int) string {
	if email == "" {
		return ""
	}
	if maxLen <= 0 {
		return ""
	}

	atIndex := strings.Index(email, "@")
	var localPart string
	if atIndex == -1 {
		localPart = email
	} else {
		localPart = email[:atIndex]
	}

	runes := []rune(localPart)
	if len(runes) <= maxLen {
		return localPart
	}
	return string(runes[:maxLen])
}

// getAvatarText получает текст для аватара из email
func getAvatarText(email string) string {
	text := TruncateEmail(email, AppConfig.MaxTextTruncateLength)
	if text == "" {
		return "?"
	}
	return strings.ToUpper(text)
}

func getTextBounds(text string, fontFace *truetype.Font, fontSize float64) (width, height int, err error) {
	opts := truetype.Options{
		Size: fontSize,
		DPI:  72,
	}
	face := truetype.NewFace(fontFace, &opts)
	defer face.Close()

	var totalWidth int
	for _, r := range text {
		advance, ok := face.GlyphAdvance(r)
		if !ok {
			continue
		}
		totalWidth += int(advance.Round())
	}

	height = int(fontSize) + int(fontSize/4)

	return totalWidth, height, nil
}

func GenerateAvatar(email string, outputPath string, customSize ...int) (string, error) {
	size := AppConfig.AvatarWidth
	if len(customSize) > 0 && customSize[0] > 0 {
		size = customSize[0]
	}

	fontFace, err := getFont(AppConfig.ImageOutFontPath)
	if err != nil {
		return "", err
	}

	bgColorInfo, err := GetRandomColorSafe()
	if err != nil {
		if len(colorsList) > 0 {
			bgColorInfo = colorsList[0]
		} else {
			bgColorInfo = ColorInfo{R: 52, G: 152, B: 219, Hex: "#3498DB"}
		}
	}

	bgColor := color.RGBA{
		R: uint8(bgColorInfo.R),
		G: uint8(bgColorInfo.G),
		B: uint8(bgColorInfo.B),
		A: 255,
	}

	textColor := GetContrastColor(bgColor)

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, bgColor)
		}
	}

	// Используем TruncateEmail для получения текста
	text := getAvatarText(email)
	fontSize := float64(AppConfig.AvatarFontSize)

	textWidth, textHeight, err := getTextBounds(text, fontFace, fontSize)
	if err != nil {
		textWidth = int(fontSize) * len([]rune(text)) / 2
		textHeight = int(fontSize)
	}

	x := (size - textWidth) / 2
	y := (size + textHeight) / 2

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(fontFace)
	c.SetFontSize(fontSize)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(textColor))

	point := freetype.Pt(x, y)
	_, err = c.DrawString(text, point)
	if err != nil {
		return "", fmt.Errorf("ошибка отрисовки текста: %w", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("ошибка создания файла: %w", err)
	}
	defer outFile.Close()

	err = png.Encode(outFile, img)
	if err != nil {
		return "", fmt.Errorf("ошибка кодирования PNG: %w", err)
	}

	return bgColorInfo.Hex, nil
}

func (c *ColorInfo) IsLight() bool {
	brightness := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
	return brightness > 128
}

func GenerateAvatarWithColor(email, outputPath, hexColor string, fontSize int, customSize ...int) (string, error) {
	size := AppConfig.AvatarWidth
	if len(customSize) > 0 && customSize[0] > 0 {
		size = customSize[0]
	}

	if fontSize <= 0 {
		fontSize = AppConfig.AvatarFontSize
	}

	fontFace, err := getFont(AppConfig.ImageOutFontPath)
	if err != nil {
		return "", err
	}

	bgColorInfo, err := GetColorByHex(hexColor)
	if err != nil {
		return "", err
	}

	bgColor := color.RGBA{
		R: uint8(bgColorInfo.R),
		G: uint8(bgColorInfo.G),
		B: uint8(bgColorInfo.B),
		A: 255,
	}

	textColor := GetContrastColor(bgColor)

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, bgColor)
		}
	}

	// Используем TruncateEmail для получения текста
	text := getAvatarText(email)
	fontSizeFloat := float64(fontSize)

	textWidth, textHeight, err := getTextBounds(text, fontFace, fontSizeFloat)
	if err != nil {
		textWidth = int(fontSizeFloat) * len([]rune(text)) / 2
		textHeight = int(fontSizeFloat)
	}

	x := (size - textWidth) / 2
	y := (size + textHeight) / 2

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(fontFace)
	c.SetFontSize(fontSizeFloat)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(textColor))

	point := freetype.Pt(x, y)
	_, err = c.DrawString(text, point)
	if err != nil {
		return "", fmt.Errorf("ошибка отрисовки текста: %w", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("ошибка создания файла: %w", err)
	}
	defer outFile.Close()

	err = png.Encode(outFile, img)
	if err != nil {
		return "", fmt.Errorf("ошибка кодирования PNG: %w", err)
	}

	return bgColorInfo.Hex, nil
}
