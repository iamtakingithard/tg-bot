package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type telegramBot struct {
	api             *tgbotapi.BotAPI
	uploadDir       string
	requestSize     int64
	chatID          int64
	uploadTransient bool
}

func (tgbot *telegramBot) uploadFile(w http.ResponseWriter, r *http.Request) {
	var err error

	defer func() {
		if err != nil {
			bs := ([]byte)(err.Error())

			//w.Header().Set("content-type", "text/plain")
			w.Header().Set("content-length", strconv.Itoa(len(bs)))
			w.WriteHeader(http.StatusBadRequest)

			_, _ = w.Write(bs)
		}
	}()

	err = r.ParseMultipartForm(tgbot.requestSize)
	if err != nil {
		return
	}

	message := r.FormValue("message")

	f, fh, err := r.FormFile("file")
	if err != nil {
		return
	}

	defer func() {
		_ = f.Close()
	}()

	if !tgbot.uploadTransient {
		err = tgbot.saveFile(f, fh.Filename)
		if err != nil {
			return
		}
	}

	doc := tgbotapi.NewDocument(tgbot.chatID, tgbotapi.FileReader{
		Name:   fh.Filename,
		Reader: f,
	})

	doc.Caption = message

	_, err = tgbot.api.Send(doc)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (tgbot *telegramBot) saveFile(mf multipart.File, filename string) (err error) {
	persist := filepath.Join(tgbot.uploadDir, filepath.Base(filename))

	f, err := os.Open(persist)
	if err != nil {
		return
	}

	defer func() {
		_ = f.Close()
	}()

	_, err = io.Copy(f, mf)
	if err != nil {
		return
	}

	_, err = mf.Seek(0, io.SeekStart)

	return
}

func main() {
	listen := os.Getenv("HTTP_LISTEN")
	if listen == "" {
		listen = ":5264"
	}

	requestSize, _ := strconv.ParseInt(os.Getenv("HTTP_REQUEST_SIZE"), 10, 64)
	if requestSize == 0 {
		requestSize = 1024 * 1024 * 50
	}

	uploadDir := os.Getenv("HTTP_UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "uploaded"
	}

	uploadTransient, _ := strconv.ParseBool(os.Getenv("HTTP_UPLOAD_TRANSIENT"))

	chatID, _ := strconv.ParseInt(os.Getenv("TELEGRAM_BOT_CHAT_ID"), 10, 64)

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	botAPI, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		panic(err)
	}

	tgbot := &telegramBot{
		api:             botAPI,
		requestSize:     requestSize,
		chatID:          chatID,
		uploadDir:       uploadDir,
		uploadTransient: uploadTransient,
	}

	http.HandleFunc("/", tgbot.uploadFile)

	err = http.ListenAndServe(listen, nil)
	if err != nil {
		panic(err)
	}
}
