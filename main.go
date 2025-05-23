package main

import (
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	db "TennisBot/database"
	ev_proc "TennisBot/event_processor"
)

var dbClient db.DBClient

func general(w http.ResponseWriter, r *http.Request) {
	log.Println("in the server")

	w.Write([]byte("response"))

}

func main() {

	log.SetFlags(log.Ldate | log.Lshortfile | log.Ltime)
	// TODO: develop for monitoring, curl http://localhost:9090/general
	http.HandleFunc("/general", general)

	go http.ListenAndServe(":9090", nil)

	/* ---------- database initialization ---------- */
	dbClient.InitDatabase()

	/* ---------- bot initialization ---------- */
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	eventProcessor := ev_proc.NewEventProcessor(bot)

	bot.Send(tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "menu", Description: "Меню бота"},
	))

	bot.Debug = false

	// TODO: get clear with these ones
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	/* ---------- event processor ---------- */

	activeRoutines := make(map[int64](chan string))

	for update := range updates {
		// TODO: run scheduler mapping == think it over == runtime lib
		go eventProcessor.Process(bot, update, activeRoutines, &dbClient)
	}
}
