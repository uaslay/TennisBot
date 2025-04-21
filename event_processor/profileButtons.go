// File: event_processor/profileButtons.go
package eventprocessor

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"errors" // Потрібен для errors.Is
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm" // Потрібен для перевірки помилок БД (errors.Is)

	db "TennisBot/database"
	ui "TennisBot/ui"
)

type fixScoreState int

const (
	awaitingOpponentUsername fixScoreState = iota // Стан очікування юзернейма
	awaitingScoreResult                           // Стан очікування вибору результату
)

// ProfileButtonHandler ... (код без змін)
func (ev_proc EventProcessor) ProfileButtonHandler(bot *tgbotapi.BotAPI, chatID int64, playerID int64, dbClient *db.DBClient) {
	player, err := dbClient.GetPlayer(playerID)
	if err != nil {
		log.Printf("ProfileButtonHandler: Помилка отримання гравця %d: %v", playerID, err)
		ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося завантажити ваш профіль."))
		return
	}

	var profileMsg tgbotapi.Chattable
	if player.AvatarFileID != "" {
		log.Printf("ProfileButtonHandler: Спроба надіслати фото для %d з FileID: %s", playerID, player.AvatarFileID)
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(player.AvatarFileID))
		photo.Caption = player.String()
		profileMsg = photo
	} else {
		log.Printf("ProfileButtonHandler: У гравця %d немає AvatarFileID, надсилаємо текст.", playerID)
		msg := tgbotapi.NewMessage(chatID, player.String())
		profileMsg = msg
	}

	if _, err := bot.Send(profileMsg); err != nil {
		log.Printf("Помилка надсилання профілю (фото/текст) гравця %d: %v", playerID, err)
		if _, ok := profileMsg.(tgbotapi.PhotoConfig); ok {
			log.Printf("ProfileButtonHandler: Фото не надіслалося, спроба надіслати текст для %d.", playerID)
			bot.Send(tgbotapi.NewMessage(chatID, player.String()))
		}
	}

	editButtons := tgbotapi.NewMessage(chatID, ui.EditMsgMenu)
	editButtons.ReplyMarkup = ui.ProfileEditButtonOption
	ev_proc.bot.Send(editButtons)
}

// ProfilePhotoEditButtonHandler ...
func (ev_proc EventProcessor) ProfilePhotoEditButtonHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), playerID int64, dbClient *db.DBClient) {
	// === ВИДАЛЕНО НЕПОТРІБНУ ЗМІННУ state ===
	// state := ui.EditPhotoRequest
	// =======================================
	player, errPlayer := dbClient.GetPlayer(playerID)
	if errPlayer != nil {
		log.Printf("ProfilePhotoEditButtonHandler: Помилка отримання гравця %d: %v", playerID, errPlayer)
		stopRoutine(playerID, activeRoutines)
		return
	}
	// Визначаємо chatID з CallbackQuery
	chatID := update.CallbackQuery.Message.Chat.ID // Змінено з From.ID на Message.Chat.ID

	if _, exists := activeRoutines[player.UserID]; exists {
		log.Printf("ProfilePhotoEditButtonHandler: Рутина вже активна для %d. Зупиняємо стару.", player.UserID)
		stopRoutine(playerID, activeRoutines)
	}

	ch := make(chan string, 1)
	activeRoutines[player.UserID] = ch

	go func(currentCh chan string) {
		timer := time.NewTimer(ui.TimerPeriod)
		defer func() {
			timer.Stop()
			mapCh, exists := activeRoutines[player.UserID]
			if exists && mapCh == currentCh {
				delete(activeRoutines, player.UserID)
				log.Printf("ProfilePhotoEditButtonHandler: Рутина для %d завершена та видалена.", player.UserID)
			} else {
				log.Printf("ProfilePhotoEditButtonHandler: Рутина для %d була замінена або вже видалена.", player.UserID)
			}
		}()

		// Надсилаємо початковий запит в рамках горутини
		ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.EditMsgPhotoRequest))
		localState := ui.EditPhotoResponse // Починаємо з очікування відповіді

		for {
			select {
			case <-timer.C:
				log.Printf("ProfilePhotoEditButtonHandler: Таймер спрацював для %d", player.UserID)
				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Час очікування сплив."))
				return
			case inputData, ok := <-currentCh:
				if !ok {
					log.Printf("ProfilePhotoEditButtonHandler: Канал для %d закрито.", player.UserID)
					return
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(ui.TimerPeriod)

				if inputData == ui.QuitChannelCommand {
					log.Printf("ProfilePhotoEditButtonHandler: Команда виходу для %d.", player.UserID)
					return
				}

				// === ВИКОРИСТОВУЄМО localState ===
				if localState == ui.EditPhotoResponse {
					fileID := inputData
					log.Printf("ProfilePhotoEditButtonHandler: Отримано FileID '%s' для оновлення гравця %d", fileID, player.UserID)

					if fileID == "" {
						log.Printf("ProfilePhotoEditButtonHandler: Отримано порожній FileID для %d.", player.UserID)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося обробити фото. Спробуйте ще раз."))
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.EditMsgPhotoRequest)) // Повторюємо запит
						continue
					}

					errUpdate := dbClient.UpdatePlayer(player.UserID, map[string]interface{}{"AvatarFileID": fileID})
					if errUpdate != nil {
						log.Printf("Помилка оновлення AvatarFileID для %d в БД: %v", player.UserID, errUpdate)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Фото отримано, але сталася помилка при оновленні профілю."))
					} else {
						log.Printf("AvatarFileID для гравця %d оновлено в БД: %s", player.UserID, fileID)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Фото профілю оновлено!"))
					}
					ev_proc.ProfileButtonHandler(bot, chatID, playerID, dbClient)
					return
				} else {
					log.Printf("ProfilePhotoEditButtonHandler: Неочікуваний стан %d для %d", localState, playerID)
					return
				}
				// ================================
			} // end select
		} // end for
	}(ch)
}

// ProfileRacketEditButtonHandler ... (код без змін)
func (ev_proc EventProcessor) ProfileRacketEditButtonHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), playerID int64, dbClient *db.DBClient) {
	state := ui.EditRacketRequest
	chatID := update.CallbackQuery.Message.Chat.ID // Використовуємо ChatID з повідомлення колбеку

	if _, exists := activeRoutines[playerID]; exists {
		log.Printf("ProfileRacketEditButtonHandler: Рутина вже активна для %d. Зупиняємо стару.", playerID)
		stopRoutine(playerID, activeRoutines)
	}

	ch := make(chan string, 1)
	activeRoutines[playerID] = ch

	go func(currentCh chan string) {
		timer := time.NewTimer(ui.TimerPeriod)
		defer func() {
			timer.Stop()
			mapCh, exists := activeRoutines[playerID]
			if exists && mapCh == currentCh {
				delete(activeRoutines, playerID)
				log.Printf("ProfileRacketEditButtonHandler: Рутина для %d завершена та видалена.", playerID)
			} else {
				log.Printf("ProfileRacketEditButtonHandler: Рутина для %d була замінена або вже видалена.", playerID)
			}
		}()

		for {
			if state == ui.EditRacketRequest {
				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.EditMsgRacketRequest))
				state = ui.EditRacketResponse
			}

			select {
			case <-timer.C:
				log.Println("ProfileRacketEditButtonHandler: timer worked")
				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Час очікування сплив."))
				return
			case inputData, ok := <-currentCh:
				if !ok {
					log.Printf("ProfileRacketEditButtonHandler: Канал для %d закрито.", playerID)
					return
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(ui.TimerPeriod)

				if inputData == ui.QuitChannelCommand {
					log.Printf("ProfileRacketEditButtonHandler: Команда виходу для %d.", playerID)
					return
				}

				if state == ui.EditRacketResponse {
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
					return
				} else {
					log.Printf("ProfileRacketEditButtonHandler: Неочікуваний стан %d для %d", state, playerID)
					return
				}
			} // end select
		} // end for
	}(ch)
}

// MyProposedGamesHandler ... (код без змін)
func (ev_proc EventProcessor) MyProposedGamesHandler(bot *tgbotapi.BotAPI, chatID int64, playerID int64, dbClient *db.DBClient) {
	myGames, err := dbClient.GetGamesByUserID(playerID)
	if err != nil {
		log.Printf("MyProposedGamesHandler: Помилка отримання ігор для %d: %v", playerID, err)
		ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося завантажити список ваших ігор."))
		return
	}

	currentTime := time.Now()
	location := currentTime.Location()
	todayStart := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, location)

	var activeGamesText strings.Builder
	activeGamesCount := 0
	activeGamesText.WriteString("📋 *Ваші активні пропозиції ігор:*\n\n")

	for _, game := range myGames {
		if game.Date == "" {
			continue
		}
		unixTimestamp, errParse := strconv.ParseInt(game.Date, 10, 64)
		if errParse != nil {
			log.Printf("MyProposedGamesHandler: Помилка парсингу дати гри %d ('%s'): %v", game.ID, game.Date, errParse)
			continue
		}
		gameTime := time.Unix(unixTimestamp, 0).In(location)

		if !gameTime.Before(todayStart) {
			activeGamesText.WriteString(fmt.Sprintf("🔹 %s (ID: %d)\n", game.String(), game.ID))
			activeGamesCount++
		} else {
			log.Printf("MyProposedGamesHandler: Гра %d (%s) є минулою, не показуємо.", game.ID, game.String())
		}
	}

	var msg tgbotapi.MessageConfig
	if activeGamesCount == 0 {
		msg = tgbotapi.NewMessage(chatID, "У вас немає активних запропонованих ігор.")
	} else {
		msg = tgbotapi.NewMessage(chatID, activeGamesText.String())
		msg.ParseMode = tgbotapi.ModeMarkdown
	}

	backButton := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад до меню", "main_menu_from_my_games")),
	)
	msg.ReplyMarkup = backButton

	if _, err := bot.Send(msg); err != nil {
		log.Printf("MyProposedGamesHandler: Помилка надсилання списку своїх ігор для %d: %v", playerID, err)
	}
}

// DeleteGames ... (код без змін)
func (ev_proc EventProcessor) DeleteGames(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), playerID int64, dbClient *db.DBClient) {
	state := ui.ListOfGames
	chatID := update.CallbackQuery.Message.Chat.ID // Використовуємо ChatID з повідомлення колбеку

	if _, exists := activeRoutines[playerID]; exists {
		log.Printf("DeleteGames: Рутина вже активна для %d. Зупиняємо стару.", playerID)
		stopRoutine(playerID, activeRoutines)
	}

	ch := make(chan string, 1)
	activeRoutines[playerID] = ch

	go func(currentCh chan string) {
		var messageID int
		timer := time.NewTimer(ui.TimerPeriod)
		defer func() {
			timer.Stop()
			mapCh, exists := activeRoutines[playerID]
			if exists && mapCh == currentCh {
				delete(activeRoutines, playerID)
				log.Printf("DeleteGames: Рутина для %d завершена та видалена.", playerID)
			} else {
				log.Printf("DeleteGames: Рутина для %d була замінена або вже видалена.", playerID)
			}
			if messageID != 0 {
				bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
			}
		}()

		for {
			resetTimer := func() {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(ui.TimerPeriod)
			}

			if state == ui.ListOfGames {
				if messageID != 0 {
					bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
					messageID = 0
				}
				games, err := dbClient.GetGamesByUserID(playerID)
				if err != nil {
					log.Printf("Помилка отримання ігор для видалення (гравець %d): %v", playerID, err)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося завантажити список ваших ігор."))
					return
				}
				var replyMarkupMainMenu tgbotapi.InlineKeyboardMarkup
				activeGamesCount := 0
				currentTime := time.Now()
				location := currentTime.Location()
				todayStart := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, location)

				for _, game := range games {
					if game.Date == "" {
						continue
					}
					unixTimestamp, errParse := strconv.ParseInt(game.Date, 10, 64)
					if errParse != nil {
						continue
					}
					gameTime := time.Unix(unixTimestamp, 0).In(location)
					if !gameTime.Before(todayStart) {
						replyMarkupMainMenu.InlineKeyboard = append(replyMarkupMainMenu.InlineKeyboard,
							tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(game.String(), fmt.Sprintf("delete_game_confirm:%d", game.ID))))
						activeGamesCount++
					}
				}
				if activeGamesCount == 0 {
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "У вас немає активних запропонованих ігор для видалення."))
					ev_proc.mainMenu(chatID)
					return
				}
				replyMarkupMainMenu.InlineKeyboard = append(replyMarkupMainMenu.InlineKeyboard,
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Скасувати", ui.QuitChannelCommand)))
				msg := tgbotapi.NewMessage(chatID, "Оберіть гру, яку бажаєте видалити:")
				msg.ReplyMarkup = replyMarkupMainMenu
				response, errSend := ev_proc.bot.Send(msg)
				if errSend != nil {
					log.Printf("Помилка надсилання списку ігор для видалення: %v", errSend)
					return
				}
				messageID = response.MessageID
				state = ui.DeleteGame
			}

			select {
			case <-timer.C:
				log.Println("DeleteGames: timer worked")
				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Час очікування сплив."))
				return
			case inputData, ok := <-currentCh:
				if !ok {
					log.Printf("DeleteGames: Канал для %d закрито.", playerID)
					return
				}
				resetTimer()

				if inputData == ui.QuitChannelCommand {
					log.Printf("DeleteGames: Команда виходу для %d.", playerID)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Скасовано."))
					ev_proc.mainMenu(chatID)
					return
				}

				if state == ui.DeleteGame {
					if strings.HasPrefix(inputData, "delete_game_confirm:") {
						gameID_str := strings.TrimPrefix(inputData, "delete_game_confirm:")
						gameID_uint64, err := strconv.ParseUint(gameID_str, 10, 64)
						if err != nil {
							log.Printf("DeleteGames: Невірний callback '%s': %v", inputData, err)
							continue
						}
						gameID := uint(gameID_uint64)

						gameToDelete, errGet := dbClient.GetGame(gameID)
						if errGet != nil || gameToDelete.Player.UserID != playerID {
							errMsg := "Помилка перевірки гри."
							if errors.Is(errGet, gorm.ErrRecordNotFound) {
								errMsg = "Цю гру вже видалено або не знайдено."
							} else if gameToDelete.Player.UserID != playerID && errGet == nil {
								errMsg = "Це не ваша гра."
							} else {
								log.Printf("Помилка отримання гри %d для видалення: %v", gameID, errGet)
							}
							ev_proc.bot.Send(tgbotapi.NewMessage(chatID, errMsg))
							state = ui.ListOfGames
							continue
						}
						responses, errResp := dbClient.GetGameResponsesByGameID(gameID)
						if errResp != nil {
							log.Printf("DeleteGames: Помилка отримання відгуків для гри %d перед видаленням: %v", gameID, errResp)
						}
						for _, resp := range responses {
							responderChatID := resp.Responder.UserID
							if responderChatID == 0 {
								log.Printf("DeleteGames: Could not get UserID for responder ID %d", resp.ResponderID)
								continue
							}
							// Визначаємо текст повідомлення ПЕРЕД використанням
							msgText := fmt.Sprintf("🚫 Гру '%s', запропоновану гравцем %s, на яку ви відгукувалися, було видалено автором.",
								gameToDelete.String(), gameToDelete.Player.NameSurname) // Потрібно отримати NameSurname пропозера, якщо gameToDelete містить Player

							ev_proc.sendMessage(tgbotapi.NewMessage(responderChatID, msgText)) // Тепер msgText визначено
						}
						errDelete := dbClient.DeleteGame(gameID)
						if errDelete != nil && !errors.Is(errDelete, gorm.ErrRecordNotFound) {
							log.Printf("Помилка видалення гри %d: %v", gameID, errDelete)
							ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося видалити гру."))
							state = ui.ListOfGames
							continue
						}
						deletedCount, errDelResp := dbClient.DeleteGameResponsesByGameID(gameID)
						if errDelResp != nil {
							log.Printf("DeleteGames: Помилка видалення GameResponses для гри %d: %v", gameID, errDelResp)
						} else {
							log.Printf("DeleteGames: Видалено %d GameResponse записів для гри %d.", deletedCount, gameID)
						}
						log.Printf("Гра %d видалена користувачем %d", gameID, playerID)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Гру видалено."))
						state = ui.ListOfGames
					} else {
						log.Printf("DeleteGames: Отримано несподівані дані '%s' у стані DeleteGame", inputData)
					}
				} // end if state == ui.DeleteGame
			} // end select
		} // end for
	}(ch)
}

// StartFixScoreFlow ... (код без змін)
func (ev_proc EventProcessor) StartFixScoreFlow(bot *tgbotapi.BotAPI, chatID int64, playerID int64, dbClient *db.DBClient, activeRoutines map[int64](chan string)) {
	// ... (перевірка на існуючу рутину) ...

	ch := make(chan string, 1)
	activeRoutines[playerID] = ch
	// Запускаємо горутину обробки
	go ev_proc.handleFixScoreRoutine(bot, chatID, playerID, dbClient, activeRoutines, ch)

	// Формуємо повідомлення З КНОПКОЮ СКАСУВАННЯ
	msg := tgbotapi.NewMessage(chatID, "З ким ви грали? Введіть @username суперника:")
	cancelKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			// Ця кнопка надішле колбек "cancel_fix_score", який рутина вже вміє обробляти
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Скасувати", "cancel_fix_score"),
		),
	)
	msg.ReplyMarkup = cancelKeyboard // Додаємо клавіатуру до повідомлення

	// Надсилаємо повідомлення
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Помилка надсилання запиту username суперника: %v", err)
		// Якщо не вдалося надіслати, рутину треба зупинити
		stopRoutine(playerID, activeRoutines) // Використовуємо stopRoutine для коректного закриття
	}
}

// handleFixScoreRoutine ...
func (ev_proc EventProcessor) handleFixScoreRoutine(bot *tgbotapi.BotAPI, chatID int64, playerID int64, dbClient *db.DBClient, activeRoutines map[int64](chan string), ch chan string) {
	currentState := awaitingOpponentUsername
	var opponentID int64
	var opponentUsername string
	var messageID int

	timer := time.NewTimer(ui.TimerPeriod * 2)
	defer func() {
		timer.Stop()
		currentCh, exists := activeRoutines[playerID]
		// --- ВИПРАВЛЕНО: Переформатовано if/else if/else для ясності та відповідності gofmt ---
		if exists && currentCh == ch {
			close(ch)
			delete(activeRoutines, playerID)
			log.Printf("handleFixScoreRoutine: Рутина для %d завершена.", playerID)
		} else if exists {
			log.Printf("handleFixScoreRoutine: Рутина для %d була замінена іншою, не видаляємо.", playerID)
		} else {
			log.Printf("handleFixScoreRoutine: Рутина для %d вже була видалена.", playerID)
		}
		// --- КІНЕЦЬ ВИПРАВЛЕННЯ ---
		if messageID != 0 {
			bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
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
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(ui.TimerPeriod * 2)

			if inputData == "cancel_fix_score" || inputData == ui.QuitChannelCommand {
				log.Printf("handleFixScoreRoutine: Фіксацію рахунку скасовано користувачем %d.", playerID)
				bot.Send(tgbotapi.NewMessage(chatID, "Фіксацію рахунку скасовано."))
				ev_proc.mainMenu(chatID)
				return
			}
			switch currentState {
			case awaitingOpponentUsername:
				opponentUsername = strings.TrimPrefix(inputData, "@")
				if opponentUsername == "" || strings.ContainsAny(opponentUsername, " \t\n") {
					bot.Send(tgbotapi.NewMessage(chatID, "Будь ласка, введіть коректний @username суперника (без пробілів):"))
					continue
				}
				opponent, err := dbClient.GetPlayerByUsername("@" + opponentUsername)
				if err != nil {
					log.Printf("Фіксація рахунку: Гравець @%s не знайдений. Помилка: %v", opponentUsername, err)
					if errors.Is(err, gorm.ErrRecordNotFound) {
						bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Гравець @%s не знайдений у базі...", opponentUsername)))
					} else {
						bot.Send(tgbotapi.NewMessage(chatID, "Сталася помилка при пошуку гравця."))
					}
					bot.Send(tgbotapi.NewMessage(chatID, "Введіть @username суперника ще раз або скасуйте."))
					continue
				}
				opponentID = opponent.UserID
				if opponentID == playerID {
					bot.Send(tgbotapi.NewMessage(chatID, "Ви не можете зафіксувати рахунок гри з самим собою :) Введіть @username суперника:"))
					continue
				}
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Гравець @%s знайдений (%s).\nЯкий результат вашої гри?", opponentUsername, opponent.NameSurname))
				callbackWin := fmt.Sprintf("fix_score_result:%d:1", opponentID)
				callbackLoss := fmt.Sprintf("fix_score_result:%d:0", opponentID)
				// --- ВИПРАВЛЕНО: Додано коми в кінці кожного рядка KeyboardRow ---
				resultKeyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Я виграв ✅", callbackWin), tgbotapi.NewInlineKeyboardButtonData("Я програв ❌", callbackLoss)), // Додано кому
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Скасувати", "cancel_fix_score")),                                                           // Додано кому
				)
				// --- КІНЕЦЬ ВИПРАВЛЕННЯ ---
				msg.ReplyMarkup = resultKeyboard
				sentMsg, errSend := bot.Send(msg)
				if errSend != nil {
					log.Printf("Помилка надсилання запиту результату гри: %v", errSend)
					return
				}
				messageID = sentMsg.MessageID
				currentState = awaitingScoreResult
			case awaitingScoreResult:
				// Цей блок тепер синтаксично коректний після виправлення попередніх помилок
				if !strings.HasPrefix(inputData, "fix_score_result:") {
					log.Printf("handleFixScoreRoutine: Отримано несподівані дані '%s' у стані awaitingScoreResult", inputData)
					bot.Send(tgbotapi.NewMessage(chatID, "Будь ласка, оберіть результат гри за допомогою кнопок вище."))
					// Немає 'else', тому помилки 'expected statement, found else' тут бути не могло,
					// вони, ймовірно, стосувалися інших місць або були фантомними.
				}
				// Якщо дані *мають* префікс "fix_score_result:", вони будуть оброблені
				// в наступній ітерації циклу Process (у секції обробки колбеків без активної рутини),
				// оскільки ця рутина не обробляє сам результат, а лише запитує його.
				// Тому тут більше нічого робити не потрібно.
			}
		}
	}
}

// --- ВИДАЛЕНО: Стара функція ScoreSubmitButtonHandler ---
// func (ev_proc EventProcessor) ScoreSubmitButtonHandler(...) { ... }
