package eventprocessor

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	db "TennisBot/database"
	ui "TennisBot/ui"
)

// StopProcessing is a string constant
// PhotoFolderPath is a string constant
const (
	StopProcessing  = "quit"
	PhotoFolderPath = "/resources/avatarPhoto/"
)

// EventProcessor is responsible for processing incoming events from the Telegram bot.
type EventProcessor struct {
	bot *tgbotapi.BotAPI
}

// Event is a struct that represents an event.
type Event struct {
	ChatID int64
	Msg    string
}

// NewEventProcessor is a constructor for the EventProcessor struct.
func NewEventProcessor(bot *tgbotapi.BotAPI) EventProcessor {
	return EventProcessor{bot: bot}
}

// TODO: error management

// Process is a method that processes incoming events from the Telegram bot.
func (ev_proc EventProcessor) Process(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), dbClient *db.DBClient) {
	if update.Message != nil {
		playerID := update.Message.From.ID
		chatID := update.Message.Chat.ID

		if update.Message.IsCommand() {
			ev_proc.processCommand(bot, update.Message.Command(), update, activeRoutines, dbClient)
		}

		if update.Message.Contact != nil {
			// log.Println("+" + update.Message.Contact.PhoneNumber)
			if activeRoutines[playerID] != nil {
				activeRoutines[playerID] <- update.Message.Contact.PhoneNumber + ":" + update.Message.From.UserName
			}
		}

		if len(update.Message.Photo) > 0 {
			if activeRoutines[playerID] != nil {
				activeRoutines[playerID] <- update.Message.Photo[len(update.Message.Photo)-1].FileID
			}
			return
		}

		switch update.Message.Text {
		case ui.ProfileButton:
			stopRoutine(playerID, activeRoutines)
			if dbClient.CheckPlayerRegistration(playerID) {
				ev_proc.ProfileButtonHandler(bot, chatID, playerID, dbClient)
			} else {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)
			}
		case ui.SingleGame:
			stopRoutine(playerID, activeRoutines)
			if dbClient.CheckPlayerRegistration(playerID) {
				ev_proc.OneTimeGameHandler(bot, update, activeRoutines, playerID, dbClient)
			} else {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)
			}
		case ui.GeneralRatingButton:
			stopRoutine(playerID, activeRoutines)  // Зупиняємо поточний процес
			if dbClient.CheckPlayerRegistration(playerID) {
				rating := ui.GetPlayerRating(fmt.Sprintf("%d", playerID))  // Отримуємо рейтинг гравця
				msg := tgbotapi.NewMessage(chatID, rating)
				bot.Send(msg)
			} else {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)  // Якщо не зареєстрований – реєстрація
			}	
		default:
			if activeRoutines[playerID] != nil {
				activeRoutines[playerID] <- update.Message.Text
			}
		}
	} else if update.CallbackQuery != nil {
		playerID := update.CallbackQuery.From.ID

		switch update.CallbackQuery.Data {
		case ui.EditOptionPhoto:
			if activeRoutines[playerID] != nil {
				stopRoutine(playerID, activeRoutines)
			}
			ev_proc.ProfilePhotoEditButtonHandler(bot, update, activeRoutines, playerID, dbClient)
		case ui.EditOptionRacket:
			if activeRoutines[playerID] != nil {
				stopRoutine(playerID, activeRoutines)
			}
			ev_proc.ProfileRacketEditButtonHandler(bot, update, activeRoutines, playerID, dbClient)
		case ui.DeleteGames:
			if activeRoutines[playerID] != nil {
				stopRoutine(playerID, activeRoutines)
			}
			ev_proc.DeleteGames(bot, update, activeRoutines, playerID, dbClient)
		case ui.EnterGameScore:
			if activeRoutines[playerID] != nil {
				stopRoutine(playerID, activeRoutines)
			}
			ev_proc.EnterGameScore(bot, update, activeRoutines, playerID, dbClient)
		default:
			input := strings.Split(update.CallbackQuery.Data, ":")
			switch input[0] {
			case ui.GameConfirmationYes:
				// log.Println("here", input)
				gameID, err := strconv.ParseUint(input[2], 10, 64)
				if err != nil {
					log.Panic(err)
				}
				game := dbClient.GetGame(uint(gameID))

				partnerID, err := strconv.ParseInt(input[1], 10, 64)
				if err != nil {
					log.Panic(err)
				}

				// FIXME: use send msg function
				msg := tgbotapi.NewMessage(partnerID, "Гра підтверджена:\n"+game.String()+"\nКонтакт гравця:")
				if _, err := bot.Send(msg); err != nil {
					log.Panic(err)
				}

				player := dbClient.GetPlayer(playerID)

				msgPlayerDetails := tgbotapi.NewPhoto(partnerID, tgbotapi.FilePath(player.AvatarPhotoPath))
				msgPlayerDetails.Caption = player.String()

				if _, err := bot.Send(msgPlayerDetails); err != nil {
					log.Panic(err)
				}
			}
		}
		// log.Println("here1", update.CallbackQuery.Data)
		if activeRoutines[playerID] != nil {
			// log.Println("here2", update.CallbackQuery.Data)

			activeRoutines[playerID] <- update.CallbackQuery.Data
		}
	}
}
