package eventprocessor

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	ui "TennisBot/ui"
)

// TODO: error management
func (ev_proc EventProcessor) mainMenu(chatID int64) {
	// TODO: message to const	ant
	msg := tgbotapi.NewMessage(chatID, "ОпціЇ головного меню:")
	msg.ReplyMarkup = ui.MainKeyboard
	if _, err := ev_proc.bot.Send(msg); err != nil {
		log.Printf("Помилка надсилання головного меню користувачу %d: %v", chatID, err)
	}
}
