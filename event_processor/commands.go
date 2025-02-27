package eventprocessor

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	db "TennisBot/database"
	ui "TennisBot/ui"
)

func (ev_proc EventProcessor) processCommand(bot *tgbotapi.BotAPI, command string, update tgbotapi.Update, activeRoutines map[int64](chan string), dbClient *db.DBClient) {
	playerID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	if command != "" {
		if command == ui.MenuCommand {
			stopRoutine(playerID, activeRoutines)
			if !dbClient.CheckPlayerRegistration(playerID) {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)
			}

			ev_proc.mainMenu(chatID)
		} else if command == ui.StartCommand {
			stopRoutine(playerID, activeRoutines)
			if !dbClient.CheckPlayerRegistration(playerID) {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)
			}

			ev_proc.mainMenu(chatID)
		}
	}
}
