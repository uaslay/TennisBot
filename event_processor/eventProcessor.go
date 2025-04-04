package eventprocessor

import (
	"fmt"
	"log"
	"regexp"
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

// isMatchMessage перевіряє, чи містить повідомлення дані матчу.
func isMatchMessage(message string) bool {
	// Регулярний вираз для знаходження рахунку (наприклад, 6-3, 4-6)
	re := regexp.MustCompile(`\d{1,2}[-:]\d{1,2}(,\s*\d{1,2}[-:]\d{1,2})*`)
	match := re.FindString(message)
	if match == "" {
		return false
	}
	return true
}

// parseMatchData витягує імена гравців і рахунок з повідомлення.
func parseMatchData(message string) (playerA, playerB, score string, err error) {
	// Приклад простого парсингу: "PlayerA vs PlayerB 6-3, 4-6"
	parts := strings.Split(message, " ")
	if len(parts) < 4 {
		return "", "", "", fmt.Errorf("недостатньо даних у повідомленні")
	}
	playerA = parts[0]
	playerB = parts[2]
	score = strings.Join(parts[3:], " ")
	return playerA, playerB, score, nil
}

// processMatchResult обробляє результат матчу.
func processMatchResult(playerA, playerB, score string, dbClient *db.DBClient) {
    // Реалізуйте логіку обробки матчу тут
    log.Printf("Processing match result: %s vs %s, score: %s", playerA, playerB, score)
}


// ProcessIncomingMessage обробляє вхідне повідомлення.
func (ev_proc EventProcessor) ProcessIncomingMessage(update tgbotapi.Update, dbClient *db.DBClient) {
	if update.Message == nil {
		return
	}
	messageText := update.Message.Text
	log.Printf("Отримано повідомлення: %s", messageText)
	if isMatchMessage(messageText) {
		playerA, playerB, score, err := parseMatchData(messageText)
		if err != nil {
			log.Println("Помилка парсингу повідомлення:", err)
			return
		}
		log.Printf("Розпізнано матч: %s vs %s, рахунок: %s", playerA, playerB, score)
		// Викликаємо обробку матчу
		go processMatchResult(playerA, playerB, score, dbClient)
	} else {
		log.Println("Повідомлення не містить даних матчу")
	}
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
		case ui.FixScoreButton:
			stopRoutine(playerID, activeRoutines) // Зупиняємо інші процеси
		
			if dbClient.CheckPlayerRegistration(playerID) {
				players := ui.LoadPlayers()
				player := players[fmt.Sprintf("%d", playerID)]
		
				// Перевіряємо, чи є у гравця активні матчі
				if len(player.ActiveMatches) == 0 {
					bot.Send(tgbotapi.NewMessage(chatID, "У вас немає активних матчів. Будь ласка, узгодьте гру з суперником перед фіксацією результату."))
					return
				}
		
				// Якщо матчі є, запитуємо суперника
				msg := tgbotapi.NewMessage(chatID, "З ким ти грав? Введи @юзернейм суперника.")
				bot.Send(msg)
				activeRoutines[playerID] = make(chan string, 1) // Чекаємо відповідь користувача
				go HandleFixScore(bot, chatID, playerID, dbClient, activeRoutines)
		
			} else {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)
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
		// case ui.EnterGameScore:
		// 	if activeRoutines[playerID] != nil {
		// 		stopRoutine(playerID, activeRoutines)
		// 	}
		// 	ev_proc.EnterGameScore(bot, update, activeRoutines, playerID, dbClient)
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
