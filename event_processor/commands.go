package eventprocessor

import (
	"log"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	db "TennisBot/database"
	ui "TennisBot/ui"
)

// Обробляє команду /menu
func (ev_proc EventProcessor) processCommand(bot *tgbotapi.BotAPI, command string, update tgbotapi.Update, activeRoutines map[int64](chan string), dbClient *db.DBClient, isRegistered bool) {
    // ... тіло функції залишається тим самим або оновлюється для використання isRegistered ...
     playerID := update.Message.From.ID // Отримуємо ID
     chatID := update.Message.Chat.ID

     if command == ui.MenuCommand || command == ui.StartCommand {
        // Використовуємо переданий isRegistered замість повторної перевірки
        if !isRegistered {
            if command == ui.StartCommand { // /start завжди показує меню або починає реєстрацію
                 ev_proc.mainMenu(chatID) // Покажемо меню, а Process подбає про реєстрацію
            } else { // /menu від незареєстрованого
                log.Printf("processCommand: /menu called by unregistered user %d. Registration should be in progress.", playerID)
                // Нічого не робимо, реєстрація триває
            }
        } else {
            // Користувач зареєстрований, показуємо головне меню
            ev_proc.mainMenu(chatID)
        }
     }	 
}
