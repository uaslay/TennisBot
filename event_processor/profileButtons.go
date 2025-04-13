// File: event_processor/profileButtons.go
package eventprocessor

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm" // Потрібен для перевірки помилок БД (errors.Is)
	"errors"     // Потрібен для errors.Is

	db "TennisBot/database"
	ui "TennisBot/ui"
)

type fixScoreState int

const (
	awaitingOpponentUsername fixScoreState = iota // Стан очікування юзернейма
	awaitingScoreResult                     // Стан очікування вибору результату
)


// ProfileButtonHandler ... (код без змін з попереднього кроку)
func (ev_proc EventProcessor) ProfileButtonHandler(bot *tgbotapi.BotAPI, chatID int64, playerID int64, dbClient *db.DBClient) {
	player, err := dbClient.GetPlayer(playerID)
	if err != nil {
		log.Printf("ProfileButtonHandler: Помилка отримання гравця %d: %v", playerID, err)
		ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося завантажити ваш профіль."))
		return
	}

	var profileMsg tgbotapi.Chattable
	if player.AvatarFileID != "" { // Перевіряємо AvatarFileID
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(player.AvatarFileID)) // Використовуємо FileID
		photo.Caption = player.String()
		profileMsg = photo
	} else {
		msg := tgbotapi.NewMessage(chatID, player.String())
		profileMsg = msg
	}

	if _, err := bot.Send(profileMsg); err != nil {
		log.Printf("Помилка надсилання профілю гравця %d: %v", playerID, err)
		if _, ok := profileMsg.(tgbotapi.PhotoConfig); ok { // Спробувати надіслати текст, якщо фото не вдалося
			bot.Send(tgbotapi.NewMessage(chatID, player.String()))
		}
	}

	editButtons := tgbotapi.NewMessage(chatID, ui.EditMsgMenu)
	editButtons.ReplyMarkup = ui.ProfileEditButtonOption
	ev_proc.bot.Send(editButtons)
}


// ProfilePhotoEditButtonHandler ... (код без змін з попереднього кроку)
func (ev_proc EventProcessor) ProfilePhotoEditButtonHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), playerID int64, dbClient *db.DBClient) {
	state := ui.EditPhotoRequest
	player, errPlayer := dbClient.GetPlayer(playerID)
	if errPlayer != nil {
		log.Printf("ProfilePhotoEditButtonHandler: Помилка отримання гравця %d: %v", playerID, errPlayer)
		stopRoutine(playerID, activeRoutines)
		return
	}
	chatID := update.CallbackQuery.From.ID

	if _, exists := activeRoutines[player.UserID]; exists {
		log.Printf("ProfilePhotoEditButtonHandler: Рутина вже активна для %d.", player.UserID)
		stopRoutine(playerID, activeRoutines)
	}

	activeRoutines[player.UserID] = make(chan string, 1)
	activeRoutines[player.UserID] <- update.CallbackQuery.Data

	timer := time.NewTimer(ui.TimerPeriod)
	defer func() {
		timer.Stop()
		if ch, ok := activeRoutines[player.UserID]; ok {
			close(ch)
			delete(activeRoutines, player.UserID)
			log.Printf("ProfilePhotoEditButtonHandler: Рутина для %d завершена.", player.UserID)
		}
	}()

	for {
		select {
		case <-timer.C:
				log.Printf("ProfilePhotoEditButtonHandler: Таймер спрацював для %d", player.UserID)
				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Час очікування сплив."))
				return // Вихід з функції
		case inputData, ok := <-activeRoutines[player.UserID]:
				if !ok {
						log.Printf("ProfilePhotoEditButtonHandler: Канал для %d закрито.", player.UserID)
						return
				}
				if !timer.Stop() {
						select { case <-timer.C: default: }
				}
				timer.Reset(ui.TimerPeriod)

				if inputData == ui.QuitChannelCommand {
						log.Printf("ProfilePhotoEditButtonHandler: Команда виходу для %d.", player.UserID)
						return
				}

				switch state {
				case ui.EditPhotoRequest:
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.EditMsgPhotoRequest))
					state = ui.EditPhotoResponse
				case ui.EditPhotoResponse:
					fileID := inputData // Отримали FileID
				
					// Оновлюємо FileID в БД
					errUpdate := dbClient.UpdatePlayer(player.UserID, map[string]interface{}{"AvatarFileID": fileID})
					if errUpdate != nil {
						log.Printf("Помилка оновлення AvatarFileID для %d: %v", player.UserID, errUpdate)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Фото отримано, але сталася помилка при оновленні профілю."))
					} else {
						log.Printf("FileID фото для гравця %d оновлено: %s", player.UserID, fileID)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Фото профілю оновлено!"))
					}
					// Показуємо оновлений профіль
					ev_proc.ProfileButtonHandler(bot, chatID, playerID, dbClient)
					return // Завершуємо рутину
				}
		}
	}
}

// ProfileRacketEditButtonHandler ... (код без змін з попереднього кроку)
func (ev_proc EventProcessor) ProfileRacketEditButtonHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), playerID int64, dbClient *db.DBClient) {
	state := ui.EditRacketRequest
	chatID := update.CallbackQuery.From.ID

	if _, exists := activeRoutines[playerID]; exists {
		log.Printf("ProfileRacketEditButtonHandler: Рутина вже активна для %d.", playerID)
		stopRoutine(playerID, activeRoutines)
	}

	activeRoutines[playerID] = make(chan string, 1)
	activeRoutines[playerID] <- update.CallbackQuery.Data

	timer := time.NewTimer(ui.TimerPeriod)
	defer func() {
		timer.Stop()
		if ch, ok := activeRoutines[playerID]; ok {
			close(ch)
			delete(activeRoutines, playerID)
			log.Printf("ProfileRacketEditButtonHandler: Рутина для %d завершена.", playerID)
		}
	}()

out:
	for {
		select {
		case <-timer.C:
			log.Println("ProfileRacketEditButtonHandler: timer worked")
			ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Час очікування сплив."))
			break out
		case inputData, ok := <-activeRoutines[playerID]:
			if !ok {
				log.Printf("ProfileRacketEditButtonHandler: Канал для %d закрито.", playerID)
				break out
			}
			if !timer.Stop() {
					select { case <-timer.C: default: }
			}
			timer.Reset(ui.TimerPeriod)

			if inputData == ui.QuitChannelCommand {
				log.Printf("ProfileRacketEditButtonHandler: Команда виходу для %d.", playerID)
				break out
			}

			switch state {
			case ui.EditRacketRequest:
				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.EditMsgRacketRequest))
				state = ui.EditRacketResponse
			case ui.EditRacketResponse:
				racketInfo := inputData
				err := dbClient.UpdatePlayer(playerID, map[string]interface{}{"Racket": racketInfo})
				if err != nil {
					log.Printf("Помилка оновлення ракетки для %d: %v", playerID, err)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося оновити інформацію про ракетку."))
				} else {
					log.Printf("Ракетка для гравця %d оновлена: %s", playerID, racketInfo)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Інформацію про ракетку оновлено!"))
					ev_proc.ProfileButtonHandler(bot, chatID, playerID, dbClient)
				}
				break out // Завершуємо
			}
		}
	}
}


// DeleteGames ... (код без змін з попереднього кроку)
func (ev_proc EventProcessor) DeleteGames(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), playerID int64, dbClient *db.DBClient) {
	state := ui.ListOfGames
	chatID := update.CallbackQuery.From.ID

	if _, exists := activeRoutines[playerID]; exists {
		log.Printf("DeleteGames: Рутина вже активна для %d.", playerID)
		stopRoutine(playerID, activeRoutines)
	}

	activeRoutines[playerID] = make(chan string, 1)
	activeRoutines[playerID] <- update.CallbackQuery.Data

	timer := time.NewTimer(ui.TimerPeriod)
	defer func() {
		timer.Stop()
		if ch, ok := activeRoutines[playerID]; ok {
			close(ch)
			delete(activeRoutines, playerID)
			log.Printf("DeleteGames: Рутина для %d завершена.", playerID)
		}
	}()

	var messageID int

out:
	for {
		select {
		case <-timer.C:
			log.Println("DeleteGames: timer worked")
			ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Час очікування сплив."))
			if messageID != 0 {
				bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
			}
			break out
		case inputData, ok := <-activeRoutines[playerID]:
			if !ok {
				log.Printf("DeleteGames: Канал для %d закрито.", playerID)
				break out
			}
			if !timer.Stop() {
					select { case <-timer.C: default: }
			}
			timer.Reset(ui.TimerPeriod)

			if inputData == ui.QuitChannelCommand {
				log.Printf("DeleteGames: Команда виходу для %d.", playerID)
				if messageID != 0 {
					bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
				}
				break out
			}

			switch state {
			case ui.ListOfGames:
				if messageID != 0 {
						bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
						messageID = 0
				}

				games, err := dbClient.GetGamesByUserID(playerID)
				if err != nil {
					log.Printf("Помилка отримання ігор для видалення (гравець %d): %v", playerID, err)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося завантажити список ваших ігор."))
					break out
				}

				var replyMarkupMainMenu tgbotapi.InlineKeyboardMarkup
				if len(games) == 0 {
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "У вас немає запропонованих ігор для видалення."))
					break out
				}

				for _, game := range games {
					replyMarkupMainMenu.InlineKeyboard = append(
						replyMarkupMainMenu.InlineKeyboard,
						tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(game.String(), fmt.Sprint(game.ID))),
					)
				}
				replyMarkupMainMenu.InlineKeyboard = append(
					replyMarkupMainMenu.InlineKeyboard,
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Скасувати", ui.QuitChannelCommand)),
				)

				msg := tgbotapi.NewMessage(chatID, "Оберіть гру, яку бажаєте видалити:")
				msg.ReplyMarkup = replyMarkupMainMenu
				response, errSend := ev_proc.bot.Send(msg)
				if errSend != nil {
					log.Printf("Помилка надсилання списку ігор для видалення: %v", errSend)
					break out
				}
				messageID = response.MessageID
				state = ui.DeleteGame


			case ui.DeleteGame:
				gameID_uint64, err := strconv.ParseUint(inputData, 10, 64)
				if err != nil {
					log.Printf("DeleteGames: Невірний callback '%s': %v", inputData, err)
					continue
				}
				gameID := uint(gameID_uint64)

				gameToDelete, errGet := dbClient.GetGame(gameID)
				// Перевіряємо помилку і належність гри користувачу
				if errGet != nil || gameToDelete.UserID != playerID {
						if errors.Is(errGet, gorm.ErrRecordNotFound) {
							log.Printf("Спроба видалення неіснуючої гри %d користувачем %d", gameID, playerID)
							ev_proc.bot.Send(tgbotapi.NewMessage(chatID,"Цю гру вже видалено або не знайдено."))
						} else if gameToDelete.UserID != playerID && errGet == nil {
							log.Printf("Спроба видалення чужої гри %d користувачем %d", gameID, playerID)
							ev_proc.bot.Send(tgbotapi.NewMessage(chatID,"Це не ваша гра."))
						} else { // Інша помилка БД
							log.Printf("Помилка отримання гри %d для видалення: %v", gameID, errGet)
							ev_proc.bot.Send(tgbotapi.NewMessage(chatID,"Сталася помилка при перевірці гри."))
						}
						state = ui.ListOfGames
						activeRoutines[playerID] <- "" // Оновити список
						continue
				}

				// Видаляємо гру (і пов'язані відгуки всередині DeleteGame)
				errDelete := dbClient.DeleteGame(gameID)
				if errDelete != nil {
					log.Printf("Помилка видалення гри %d: %v", gameID, errDelete)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося видалити гру."))
				} else {
					log.Printf("Гра %d видалена користувачем %d", gameID, playerID)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Гру видалено."))
				}
				state = ui.ListOfGames
				activeRoutines[playerID] <- "" // Оновити список
			}
		}
	}
}


// StartFixScoreFlow ... (код без змін з попереднього кроку)
func (ev_proc EventProcessor) StartFixScoreFlow(bot *tgbotapi.BotAPI, chatID int64, playerID int64, dbClient *db.DBClient, activeRoutines map[int64](chan string)) {
	if _, exists := activeRoutines[playerID]; exists {
		log.Printf("StartFixScoreFlow: Рутина вже активна для %d.", playerID)
		ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Будь ласка, завершіть попередню дію (фіксація рахунку)."))
		return
	}

	ch := make(chan string, 1)
	activeRoutines[playerID] = ch

	go ev_proc.handleFixScoreRoutine(bot, chatID, playerID, dbClient, activeRoutines, ch)

	msg := tgbotapi.NewMessage(chatID, "З ким ви грали? Введіть @username суперника:")
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Помилка надсилання запиту username суперника: %v", err)
		stopRoutine(playerID, activeRoutines)
	}
}

// handleFixScoreRoutine ... (код без змін з попереднього кроку)
func (ev_proc EventProcessor) handleFixScoreRoutine(bot *tgbotapi.BotAPI, chatID int64, playerID int64, dbClient *db.DBClient, activeRoutines map[int64](chan string), ch chan string) {
	currentState := awaitingOpponentUsername
	var opponentID int64
	var opponentUsername string // Тільки ім'я без @

	timer := time.NewTimer(ui.TimerPeriod * 2)
	defer func() {
		timer.Stop()
		currentCh, exists := activeRoutines[playerID]
		if exists && currentCh == ch {
			close(ch)
			delete(activeRoutines, playerID)
			log.Printf("handleFixScoreRoutine: Рутина для %d завершена.", playerID)
		} else if exists {
			log.Printf("handleFixScoreRoutine: Рутина для %d була замінена іншою, не видаляємо.", playerID)
		} else {
			log.Printf("handleFixScoreRoutine: Рутина для %d вже була видалена.", playerID)
		}
	}()

	for {
		select {
		case <-timer.C:
			log.Printf("handleFixScoreRoutine: Таймер спрацював для %d", playerID)
			bot.Send(tgbotapi.NewMessage(chatID, "Час на фіксацію рахунку вичерпано."))
			return
		case inputData, ok := <-ch:
			if !ok {
				log.Printf("handleFixScoreRoutine: Канал для %d закрито.", playerID)
				return
			}
			if !timer.Stop() {
					select { case <-timer.C: default: }
			}
			timer.Reset(ui.TimerPeriod * 2)

			switch currentState {
			case awaitingOpponentUsername:
				opponentUsername = strings.TrimPrefix(inputData, "@")
				if opponentUsername == "" {
						bot.Send(tgbotapi.NewMessage(chatID, "Будь ласка, введіть коректний @username суперника:"))
						continue
				}

				opponent, err := dbClient.GetPlayerByUsername("@" + opponentUsername)
				if err != nil {
						log.Printf("Фіксація рахунку: Гравець @%s не знайдений. Помилка: %v", opponentUsername, err)
						// Перевіряємо, чи помилка "не знайдено"
						if errors.Is(err, gorm.ErrRecordNotFound) {
							bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Гравець @%s не знайдений у базі. Перевірте правильність написання або попросіть суперника зареєструватися (/start).", opponentUsername)))
						} else {
							bot.Send(tgbotapi.NewMessage(chatID, "Сталася помилка при пошуку гравця."))
						}
						return // Завершуємо рутину
				}
				opponentID = opponent.UserID

				if opponentID == playerID {
					bot.Send(tgbotapi.NewMessage(chatID, "Ви не можете зафіксувати рахунок гри з самим собою :) Введіть @username суперника:"))
					continue
				}

				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Гравець @%s знайдений. Який результат вашої гри?", opponentUsername))
				callbackWin := fmt.Sprintf("fix_score_result:%d:1", opponentID)
				callbackLoss := fmt.Sprintf("fix_score_result:%d:0", opponentID)
				resultKeyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Я виграв ✅", callbackWin),
						tgbotapi.NewInlineKeyboardButtonData("Я програв ❌", callbackLoss),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("⬅️ Скасувати", "cancel_fix_score"),
					),
				)
				msg.ReplyMarkup = resultKeyboard
				_, errSend := bot.Send(msg)
				if errSend != nil {
					log.Printf("Помилка надсилання запиту результату гри: %v", errSend)
					return
				}
				currentState = awaitingScoreResult

			case awaitingScoreResult:
				log.Printf("handleFixScoreRoutine: Отримано дані '%s' у стані awaitingScoreResult (має оброблятися як callback).", inputData)
				// Цей стан більше не використовується, колбек обробляється в Process
				return // Вихід з рутини
			}
		}
	}
}

// ScoreSubmitButtonHandler - ВИПРАВЛЕНО
func (ev_proc EventProcessor) ScoreSubmitButtonHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, dbClient *db.DBClient) {
	if update.CallbackQuery == nil {
		return
	}
	data := update.CallbackQuery.Data
	chatID := update.CallbackQuery.Message.Chat.ID
	callbackQueryID := update.CallbackQuery.ID // Зберігаємо ID для відповіді

	parts := strings.Split(data, ":")
	if len(parts) < 4 || parts[0] != "score" {
		log.Printf("ScoreSubmitButtonHandler: Невірний формат даних '%s'", data)
		callbackResp := tgbotapi.NewCallback(callbackQueryID, "Помилка: Невірний формат даних")
		bot.Request(callbackResp) // Відповідаємо на колбек
		// Можливо, надіслати повідомлення в чат?
		// bot.Send(tgbotapi.NewMessage(chatID, "Помилка: Невірний формат даних для фіксації рахунку."))
		return
	}

	playerAID_int64, errA := strconv.ParseInt(parts[1], 10, 64)
	if errA != nil {
		log.Printf("ScoreSubmitButtonHandler: Помилка парсингу playerAID '%s': %v", parts[1], errA)
		callbackResp := tgbotapi.NewCallback(callbackQueryID, "Помилка: Невірний ID гравця A")
		bot.Request(callbackResp)
		return
	}

	playerBID_int64, errB := strconv.ParseInt(parts[2], 10, 64)
	if errB != nil {
		log.Printf("ScoreSubmitButtonHandler: Помилка парсингу playerBID '%s': %v", parts[2], errB)
		callbackResp := tgbotapi.NewCallback(callbackQueryID, "Помилка: Невірний ID гравця B")
		bot.Request(callbackResp)
		return
	}

	result, errRes := strconv.ParseFloat(parts[3], 64)
	if errRes != nil || (result != 1.0 && result != 0.0) {
		log.Printf("ScoreSubmitButtonHandler: Помилка парсингу result '%s': %v", parts[3], errRes)
		callbackResp := tgbotapi.NewCallback(callbackQueryID, "Помилка: Невірний результат гри")
		bot.Request(callbackResp)
		return
	}

	// --- ВИПРАВЛЕННЯ: Визначаємо msg перед використанням ---
	confirmationText := "Рахунок зафіксовано. Рейтинг оновлено!" // Текст для повідомлення і колбека

	errUpdate := ui.UpdatePlayerRating(playerAID_int64, playerBID_int64, result, dbClient)
	if errUpdate != nil {
		log.Printf("ScoreSubmitButtonHandler: Помилка оновлення рейтингу: %v", errUpdate)
		confirmationText = fmt.Sprintf("Помилка при оновленні рейтингу: %v", errUpdate)
		bot.Send(tgbotapi.NewMessage(chatID, confirmationText)) // Надсилаємо помилку в чат
	} else {
		log.Printf("ScoreSubmitButtonHandler: Рейтинг оновлено для %d vs %d, результат A=%.1f", playerAID_int64, playerBID_int64, result)
		// Надсилаємо повідомлення з підтвердженням
		msg := tgbotapi.NewMessage(chatID, confirmationText) // Визначаємо msg тут
		// --- ВИПРАВЛЕННЯ: Видаляємо некоректний виклик GetPlayerRating ---
		// ratingMsgA := ui.GetPlayerRating(playerAID_int64, dbClient) // НЕ ПОТРІБНО ТУТ
		// ratingMsgB := ui.GetPlayerRating(playerBID_int64, dbClient) // НЕ ПОТРІБНО ТУТ
		// msg.Text += fmt.Sprintf("\nВаш новий стан: %s\nСтан суперника: %s", ratingMsgA, ratingMsgB)
		if _, err := bot.Send(msg); err != nil {
			log.Println("Помилка надсилання повідомлення підтвердження ScoreSubmitButtonHandler:", err)
		}
	}

	// Відповідаємо на CallbackQuery, щоб прибрати годинник у користувача
	callbackResp := tgbotapi.NewCallback(callbackQueryID, confirmationText) // Використовуємо визначений текст
	if _, err := bot.Request(callbackResp); err != nil {
			log.Printf("Помилка відповіді на callback query %s: %v", callbackQueryID, err)
	}
}


// --- ВИДАЛЕНО: Стара функція HandleFixScore ---
// func HandleFixScore(...) { ... }