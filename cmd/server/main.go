package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Sasha-pr/generator2/pkg/avatar"
)

var (
	selectedColorHex  string
	selectedFontSize  int
	selectedImageSize int
	selectedEmail     string
	tempAvatarPath    string
	uploadsDir        string
	workDir           string
)

type UpdatePreviewRequest struct {
	Email     string `json:"email"`
	Color     string `json:"color"`
	FontSize  int    `json:"font_size"`
	ImageSize int    `json:"image_size"`
}

func main() {
	// Загрузка конфигов
	err := avatar.LoadConfig("config.json")
	if err != nil {
		log.Printf("⚠️ Предупреждение: %v", err)
	}

	// Загрузка цветов с fallback
	err = avatar.LoadColors("colors.json")
	if err != nil {
		log.Printf("⚠️ Предупреждение: %v", err)
	}

	// Получаем рабочую директорию
	workDir, _ = os.Getwd()

	// Используем путь из конфига
	uploadsDir = filepath.Join(workDir, avatar.AppConfig.SaveAvatarToPath)
	err = os.MkdirAll(uploadsDir, avatar.AppConfig.DirPermission0755)
	if err != nil {
		log.Printf("Ошибка создания папки uploads: %v", err)
	}
	log.Printf("Папка для сохранения: %s", uploadsDir)

	// Инициализация шаблонов с функциями
	funcMap := template.FuncMap{
		"isLight": func(c avatar.ColorInfo) bool { return c.IsLight() },
	}

	tmplStep1 := template.Must(template.ParseFiles("web/templates/step1.html"))
	tmplStep2 := template.Must(template.New("step2.html").Funcs(funcMap).ParseFiles("web/templates/step2.html"))
	tmplStep3 := template.Must(template.ParseFiles("web/templates/step3.html"))

	// Статические файлы
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir))))

	// Маршруты
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmplStep1.Execute(w, nil)
	})

	http.HandleFunc("/step1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		selectedEmail = r.FormValue("email")
		selectedImageSize = avatar.AppConfig.AvatarWidth
		selectedFontSize = avatar.AppConfig.AvatarFontSize

		randomColor, err := avatar.GetRandomColorSafe()
		if err != nil {
			log.Printf("⚠️ Ошибка получения цвета: %v", err)
			randomColor = avatar.ColorInfo{Name: "Синий", Hex: "#3498DB", R: 52, G: 152, B: 219}
		}
		selectedColorHex = randomColor.Hex

		tempAvatarPath = filepath.Join(uploadsDir, "preview.png")
		_, err = avatar.GenerateAvatarWithColor(selectedEmail, tempAvatarPath, selectedColorHex, selectedFontSize, selectedImageSize)
		if err != nil {
			log.Printf("Ошибка генерации предпросмотра: %v", err)
		}

		tmplStep2.Execute(w, struct {
			Email             string
			Preview           string
			Colors            []avatar.ColorInfo
			SelectedColor     string
			SelectedFontSize  int
			SelectedImageSize int
			Timestamp         int64
		}{
			Email:             selectedEmail,
			Preview:           "/preview.png",
			Colors:            avatar.GetAllColors(),
			SelectedColor:     selectedColorHex,
			SelectedFontSize:  selectedFontSize,
			SelectedImageSize: selectedImageSize,
			Timestamp:         time.Now().Unix(),
		})
	})

	http.HandleFunc("/update-preview", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req UpdatePreviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tempAvatarPath = filepath.Join(uploadsDir, "preview.png")
		hexColor := req.Color
		if hexColor == "" {
			color, err := avatar.GetRandomColorSafe()
			if err != nil {
				hexColor = "#3498DB"
			} else {
				hexColor = color.Hex
			}
		}

		_, err := avatar.GenerateAvatarWithColor(req.Email, tempAvatarPath, hexColor, req.FontSize, req.ImageSize)
		if err != nil {
			log.Printf("Ошибка генерации: %v", err)
			http.Error(w, fmt.Sprintf("Ошибка генерации: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	http.HandleFunc("/step2", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			selectedColorHex = r.FormValue("color")
			selectedFontSize, _ = strconv.Atoi(r.FormValue("fontsize"))
			selectedImageSize, _ = strconv.Atoi(r.FormValue("imagesize"))

			if selectedFontSize <= 0 {
				selectedFontSize = avatar.AppConfig.AvatarFontSize
			}
			if selectedImageSize <= 0 {
				selectedImageSize = avatar.AppConfig.AvatarWidth
			}

			tempAvatarPath = filepath.Join(uploadsDir, "preview.png")
			_, err := avatar.GenerateAvatarWithColor(selectedEmail, tempAvatarPath, selectedColorHex, selectedFontSize, selectedImageSize)
			if err != nil {
				log.Printf("Ошибка генерации: %v", err)
			}

			tmplStep3.Execute(w, struct {
				Preview string
			}{
				Preview: "/preview.png",
			})
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})

	http.HandleFunc("/step3", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/step2", http.StatusSeeOther)
			return
		}

		filename := r.FormValue("filename")
		savePath := r.FormValue("filepath")

		filename = strings.TrimSpace(filename)
		if filename == "" {
			filename = fmt.Sprintf("avatar_%d.png", time.Now().Unix())
		}
		if !strings.HasSuffix(strings.ToLower(filename), ".png") {
			filename = filename + ".png"
		}

		var destPath string
		savePath = strings.TrimSpace(savePath)

		if savePath == "" || savePath == "uploads" {
			destPath = filepath.Join(uploadsDir, filename)
		} else {
			savePath = strings.ReplaceAll(savePath, "..", "")
			savePath = strings.Trim(savePath, "/\\")
			fullSavePath := filepath.Join(workDir, savePath)
			if err := os.MkdirAll(fullSavePath, avatar.AppConfig.DirPermission0755); err != nil {
				http.Error(w, fmt.Sprintf("Ошибка создания папки: %v", err), http.StatusInternalServerError)
				return
			}
			destPath = filepath.Join(fullSavePath, filename)
		}

		if _, err := os.Stat(tempAvatarPath); os.IsNotExist(err) {
			_, err := avatar.GenerateAvatarWithColor(selectedEmail, tempAvatarPath, selectedColorHex, selectedFontSize, selectedImageSize)
			if err != nil {
				http.Error(w, "Не удалось создать preview", http.StatusInternalServerError)
				return
			}
		}

		sourceFile, err := os.Open(tempAvatarPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Ошибка открытия файла: %v", err), http.StatusInternalServerError)
			return
		}
		defer sourceFile.Close()

		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, avatar.AppConfig.DirPermission0755); err != nil {
			http.Error(w, fmt.Sprintf("Ошибка создания папки: %v", err), http.StatusInternalServerError)
			return
		}

		destFile, err := os.Create(destPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Ошибка создания файла: %v", err), http.StatusInternalServerError)
			return
		}
		defer destFile.Close()

		bytesCopied, err := io.Copy(destFile, sourceFile)
		if err != nil {
			http.Error(w, fmt.Sprintf("Ошибка сохранения: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div class="success-message">
			<h3>✅ Аватар успешно сохранён!</h3>
			<p><strong>📁 Путь:</strong> %s</p>
			<p><strong>📏 Размер:</strong> %d байт</p>
			<p><strong>🎨 Цвет фона:</strong> %s</p>
			<p><strong>📐 Размер картинки:</strong> %dx%d px</p>
			<hr>
			<p>
				<a href="/uploads/%s" class="btn-primary" target="_blank">👁️ Просмотреть файл</a>
				<a href="/" class="btn-primary">✨ Создать новый</a>
			</p>
		</div>`, destPath, bytesCopied, selectedColorHex, selectedImageSize, selectedImageSize, filename)
	})

	http.HandleFunc("/preview.png", func(w http.ResponseWriter, r *http.Request) {
		previewPath := filepath.Join(uploadsDir, "preview.png")
		if _, err := os.Stat(previewPath); err == nil {
			http.ServeFile(w, r, previewPath)
		} else {
			http.Error(w, "Preview not found", http.StatusNotFound)
		}
	})

	// Создаём сервер
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      nil,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запуск сервера в горутине
	go func() {
		log.Println("========================================")
		log.Println("✅ Сервер запущен на http://localhost:8080")
		log.Println("📁 Папка для сохранения:", uploadsDir)
		log.Println("========================================")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Получен сигнал завершения, выключаем сервер...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка при выключении сервера: %v", err)
	}

	log.Println("✅ Сервер gracefully shutdown")
}
