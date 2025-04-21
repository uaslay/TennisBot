package eventprocessor

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	// "errors" // Поки не використовується
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	// "gorm.io/gorm" // Поки не використовується
	db "TennisBot/database"
	ui "TennisBot/ui"
)

// OneTimeGameHandler is a handler for one-time game
func (ev_proc EventProcessor) OneTimeGameHandler(
	bot *tgbotapi.BotAPI,
	update tgbotapi.Update,
	activeRoutines map[int64](chan string),
	playerID int64,
	dbClient *db.DBClient) {

	var messageID int                                     // ID повідомлення, яке редагуємо або видаляємо
	var replyMarkupMainMenu tgbotapi.InlineKeyboardMarkup // Клавіатура для списку ігор

	singleGameChoice := DefaultOneTimeGameChoice() // Ініціалізація дефолтного вибору

	state := ui.SingleGameMenu                        // Початковий стан
	player, errPlayer := dbClient.GetPlayer(playerID) // Отримуємо дані гравця
	if errPlayer != nil {
		log.Printf("OneTimeGameHandler: Помилка отримання гравця %d: %v", playerID, errPlayer)
		var chatID int64
		if update.Message != nil {
			chatID = update.Message.Chat.ID
		} else if update.CallbackQuery != nil {
			chatID = update.CallbackQuery.Message.Chat.ID
		}
		if chatID != 0 {
			ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Помилка завантаження даних профілю."))
		}
		return
	}

	var chatID int64 // Визначаємо chatID
	if update.Message != nil {
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		chatID = update.CallbackQuery.Message.Chat.ID
	} else {
		log.Println("OneTimeGameHandler: Не вдалося визначити chatID")
		return
	}

	if _, exists := activeRoutines[player.UserID]; exists { // Перевірка активної рутини
		log.Printf("OneTimeGameHandler: Рутина вже активна для користувача %d.", player.UserID)
		ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Будь ласка, завершіть попередню дію."))
		return
	}

	activeRoutines[player.UserID] = make(chan string, 1) // Створюємо канал

	// Надсилаємо початковий тригер
	if update.Message != nil {
		activeRoutines[player.UserID] <- update.Message.Text
	} else if update.CallbackQuery != nil {
		activeRoutines[player.UserID] <- update.CallbackQuery.Data
	}

	replyMarkup := ui.NewKeyboard() // Клавіатура для меню створення

	timer := time.NewTimer(ui.TimerPeriod) // Таймер
	defer func() {                         // Очищення при виході
		timer.Stop()
		if ch, ok := activeRoutines[player.UserID]; ok {
			close(ch)
			delete(activeRoutines, player.UserID)
			log.Printf("OneTimeGameHandler: Рутина для користувача %d завершена та видалена.", player.UserID)
		}
	}()

out:
	for {
		select {
		case <-timer.C: // Тайм-аут
			log.Printf("OneTimeGameHandler: Таймер спрацював для користувача %d.", player.UserID)
			msg := tgbotapi.NewMessage(chatID, "Час очікування сплив. Будь ласка, спробуйте ще раз.")
			if messageID != 0 {
				ev_proc.bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
			}
			_, _ = ev_proc.bot.Send(msg)
			ev_proc.mainMenu(chatID)
			break out

		case inputData, ok := <-activeRoutines[player.UserID]: // Отримано дані
			if !ok {
				log.Printf("OneTimeGameHandler: Канал для користувача %d закрито.", player.UserID)
				break out
			} // Канал закрито

			// Скидання таймера
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(ui.TimerPeriod)

			// Обробка команди виходу
			if inputData == ui.QuitChannelCommand {
				log.Printf("OneTimeGameHandler: Отримано команду виходу для користувача %d.", player.UserID)
				if messageID != 0 {
					ev_proc.bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
				}
				ev_proc.mainMenu(chatID)
				break out
			}

			// Головний switch станів
			switch state {
			case ui.SingleGameMenu: // Показ списку чужих ігор + кнопки "Мої ігри", "Запропонувати"
				if len(replyMarkupMainMenu.InlineKeyboard) > 0 {
					replyMarkupMainMenu.InlineKeyboard = nil
				}
				games, errGames := dbClient.GetGames()
				if errGames != nil { // Обробка помилки отримання ігор
					log.Printf("OneTimeGameHandler: Помилка отримання списку ігор: %v", errGames)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося завантажити список ігор."))
					ev_proc.mainMenu(chatID)
					break out
				}
				currentTime := time.Now()
				todayStart := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())
				var gameButtons [][]tgbotapi.InlineKeyboardButton
				for _, game := range games { // Фільтрація та формування кнопок чужих ігор
					if game.Date == "" {
						continue
					}
					unixTimestamp, errParse := strconv.ParseInt(game.Date, 10, 64)
					if errParse != nil {
						log.Printf("Помилка парсингу дати гри %d ('%s'): %v", game.ID, game.Date, errParse)
						continue
					}
					gameTime := time.Unix(unixTimestamp, 0)
					if (gameTime.After(todayStart) || gameTime.Equal(todayStart)) && game.Player.UserID != playerID {
						gameButtons = append(gameButtons, tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData(game.String(), fmt.Sprint(game.ID)),
						))
					}
				}
				controlButtons := tgbotapi.NewInlineKeyboardRow( // Кнопки управління
					tgbotapi.NewInlineKeyboardButtonData("🧐 Мої ігри", ui.MyProposedGamesCallback),
					tgbotapi.NewInlineKeyboardButtonData(ui.ProposeGame, ui.ProposeGame),
				)
				replyMarkupMainMenu.InlineKeyboard = append(gameButtons, controlButtons) // Збираємо клавіатуру
				msgText := ui.InitialMessage
				if len(gameButtons) == 0 {
					msgText = "Зараз немає активних пропозицій від інших гравців..."
				}
				if messageID != 0 {
					ev_proc.bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
					messageID = 0
				} // Видаляємо старе повідомлення
				msg := tgbotapi.NewMessage(chatID, msgText)
				msg.ReplyMarkup = replyMarkupMainMenu
				response, err := ev_proc.bot.Send(msg) // Надсилаємо нове
				if err != nil {
					log.Printf("OneTimeGameHandler: Помилка надсилання списку ігор: %v", err)
					break out
				}
				messageID = response.MessageID
				state = ui.ProcessSingleGameMenu // Очікуємо вибору гри або натискання кнопки

			case ui.ProcessSingleGameMenu: // Очікування вибору гри або кнопки
				if inputData == ui.ProposeGame {
					state = ui.ProposeGameMenu          // Перехід до меню створення гри
					activeRoutines[player.UserID] <- "" // Тригер для нового стану
				} else if inputData == ui.MyProposedGamesCallback {
					log.Printf("OneTimeGameHandler: Користувач %d натиснув 'Мої ігри' (обробляється в Process)", playerID)
					// Залишаємось в цьому стані, чекаємо на наступні дії або колбек з Process
				} else if inputData != "" { // Обрано гру зі списку (inputData - це gameID)
					gameID_uint64, err := strconv.ParseUint(inputData, 10, 64)
					if err != nil {
						log.Printf("OneTimeGameHandler: Невірний gameID '%s': %v", inputData, err)
						state = ui.SingleGameMenu
						activeRoutines[player.UserID] <- ""
						continue
					}
					gameID := uint(gameID_uint64)
					game, errGame := dbClient.GetGame(gameID)
					if errGame != nil {
						log.Printf("OneTimeGameHandler: Помилка отримання гри %d: %v", gameID, errGame)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Помилка завантаження гри.")) // Повідомлення користувачу
						state = ui.SingleGameMenu
						activeRoutines[player.UserID] <- ""
						continue
					}
					gamePlayer, errGamePlayer := dbClient.GetPlayer(game.Player.UserID)
					if errGamePlayer != nil {
						log.Printf("OneTimeGameHandler: Помилка отримання даних гравця %d для гри %d: %v", game.Player.UserID, gameID, errGamePlayer)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося отримати дані гравця для цієї гри."))
						state = ui.SingleGameMenu
						activeRoutines[player.UserID] <- ""
						continue
					}
					// Надсилання фото/інфо gamePlayer
					var playerInfoMsg tgbotapi.Chattable
					if gamePlayer.AvatarFileID != "" {
						photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(gamePlayer.AvatarFileID))
						photo.Caption = "Дані гравця:\n\n" + gamePlayer.String()
						playerInfoMsg = photo
					} else {
						msgInfo := tgbotapi.NewMessage(chatID, "Дані гравця:\n\n"+gamePlayer.String())
						playerInfoMsg = msgInfo
					}
					if _, errSend := bot.Send(playerInfoMsg); errSend != nil {
						log.Printf("OneTimeGameHandler: Помилка надсилання даних гравця %d: %v", gamePlayer.UserID, errSend)
						if _, okPhoto := playerInfoMsg.(tgbotapi.PhotoConfig); okPhoto { // Спроба надіслати текст
							bot.Send(tgbotapi.NewMessage(chatID, "Дані гравця:\n\n"+gamePlayer.String()))
						}
					}
					// Надсилання кнопок confirm_game:yes/no
					msgReply := tgbotapi.NewMessage(chatID, "Бажаєте відгукнутися на цю гру?")
					confirmDataYes := fmt.Sprintf("confirm_game:yes:%d", gameID)
					confirmDataNo := fmt.Sprintf("confirm_game:no:%d", gameID)
					msgReply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData(ui.GameConfirmationYes, confirmDataYes),
							tgbotapi.NewInlineKeyboardButtonData(ui.GameConfirmationNo, confirmDataNo),
						),
					)
					_, errSendReply := ev_proc.bot.Send(msgReply)
					if errSendReply != nil {
						log.Printf("OneTimeGameHandler: Помилка надсилання запиту підтвердження гри: %v", errSendReply)
					}
					// Залишаємось у стані state = ui.ProcessSingleGameMenu, чекаємо callback
				} else { // Незрозумілі дані
					log.Printf("OneTimeGameHandler: Отримано неочікувані дані '%s' у стані ProcessSingleGameMenu", inputData)
					state = ui.SingleGameMenu // Повернення до списку
					activeRoutines[player.UserID] <- ""
				}

			case ui.ProposeGameMenu: // Показ меню створення гри
				log.Printf("OneTimeGameHandler: Вхід у стан ProposeGameMenu для %d", playerID)
				var msgChattable tgbotapi.Chattable
				replyMarkup = ui.NewKeyboard()                // Скидаємо клавіатуру
				singleGameChoice = DefaultOneTimeGameChoice() // Скидаємо вибір
				if messageID != 0 {
					bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
					messageID = 0
				} // Видаляємо старе повідомлення
				msgNew := tgbotapi.NewMessage(chatID, ui.InitialMessage)
				msgNew.ReplyMarkup = replyMarkup
				msgChattable = msgNew
				response, err := ev_proc.bot.Send(msgChattable) // Надсилаємо нове
				if err != nil {
					log.Printf("OneTimeGameHandler: Помилка надсилання меню створення гри: %v", err)
					ev_proc.mainMenu(chatID)
					break out
				}
				messageID = response.MessageID
				state = ui.EditProposeGameMenu // Переходимо до редагування

			case ui.EditProposeGameMenu: // Редагування опцій гри
				log.Printf("OneTimeGameHandler: Обробка даних '%s' у стані EditProposeGameMenu для %d", inputData, playerID)
				if inputData == ui.Ok { // Натиснуто OK
					// Перевіряємо дату (має бути встановлена або встановлюємо сьогодні)
					if singleGameChoice.Date == "" || singleGameChoice.Date == ui.DateWillSpecify {
						singleGameChoice.Date = strconv.FormatInt(time.Now().Unix(), 10)
					} else if singleGameChoice.Date == ui.DateTomorrow {
						singleGameChoice.Date = strconv.FormatInt(time.Now().AddDate(0, 0, 1).Unix(), 10)
					}
					state = ui.Selected                 // Перехід до фінального підтвердження
					activeRoutines[player.UserID] <- "" // Тригер для стану Selected
				} else if inputData == ui.Back { // Натиснуто Назад
					state = ui.SingleGameMenu // Повернення до списку ігор
					if messageID != 0 {
						bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
						messageID = 0
					}
					activeRoutines[player.UserID] <- "" // Тригер для стану SingleGameMenu
				} else if inputData != "" { // Натиснуто кнопку опції
					newReplyMarkup, errChoice := processGameChoice(inputData, replyMarkup, &singleGameChoice)
					if errChoice != nil {
						log.Printf("OneTimeGameHandler: Помилка обробки вибору '%s': %v", inputData, errChoice)
						// Можливо, надіслати повідомлення користувачу? Або просто продовжити очікування.
						continue // Пропустити решту ітерації та чекати на новий інпут
					}
					replyMarkup = newReplyMarkup                                                  // Оновлюємо клавіатуру для наступного редагування
					msgEdit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, replyMarkup) // Готуємо редагування повідомлення
					_, errEdit := ev_proc.bot.Send(msgEdit)                                       // Редагуємо
					if errEdit != nil {
						log.Printf("OneTimeGameHandler: Помилка оновлення клавіатури: %v", errEdit)
					}
					// Залишаємось у цьому ж стані state = ui.EditProposeGameMenu, чекаємо наступних дій
				} else {
					log.Printf("OneTimeGameHandler: Отримано порожні дані у стані EditProposeGameMenu для %d", playerID)
				}

			case ui.Selected: // Показ фінального підтвердження гри перед збереженням
				log.Printf("OneTimeGameHandler: Вхід у стан Selected для %d, дані: %+v", playerID, singleGameChoice)
				// Фінальна перевірка даних та встановлення дефолтів
				if singleGameChoice.Area == "" {
					singleGameChoice.Area = "Не вказано"
				} else {
					singleGameChoice.Area = strings.TrimSpace(singleGameChoice.Area)
				}
				if _, err := strconv.ParseInt(singleGameChoice.Date, 10, 64); err != nil { // Перевіряємо чи дата вже timestamp
					log.Printf("OneTimeGameHandler: Некоректний формат дати '%s' перед підтвердженням. Встановлюємо сьогодні.", singleGameChoice.Date)
					singleGameChoice.Date = strconv.FormatInt(time.Now().Unix(), 10) // Встановлюємо сьогодні як fallback
				}
				if singleGameChoice.Time == ui.TimeDoNotCare || singleGameChoice.Time == ui.TimeWillSpecify {
					singleGameChoice.Time = "Неважливо"
				}
				if singleGameChoice.Court == ui.CourtDoNotCare || singleGameChoice.Court == ui.CourtWillSpecify {
					singleGameChoice.Court = "Неважливо"
				}
				// Видалення повідомлення з меню редагування
				if messageID != 0 {
					bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
					messageID = 0
				}
				// Надсилання повідомлення з підсумком та кнопками Так/Ні
				msg := tgbotapi.NewMessage(chatID, "Перевірте та підтвердіть вашу гру:")
				msg.Text = singleGameChoice.Serialize() // Формуємо текст гри
				msg.ReplyMarkup = ui.ChoiceConfirmation // Додаємо кнопки Так/Ні
				response, err := ev_proc.bot.Send(msg)
				if err != nil {
					log.Printf("Помилка надсилання повідомлення підтвердження: %v", err)
					ev_proc.mainMenu(chatID)
					break out
				}
				messageID = response.MessageID // Зберігаємо ID повідомлення з Так/Ні
				state = ui.AllSelected         // Очікування відповіді Так/Ні

			case ui.AllSelected: // Обробка відповіді Так/Ні
				log.Printf("OneTimeGameHandler: Обробка даних '%s' у стані AllSelected для %d", inputData, playerID)
				if messageID != 0 {
					bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
					messageID = 0
				} // Видаляємо повідомлення з Так/Ні
				player, errPlayer := dbClient.GetPlayer(playerID)
				if errPlayer != nil {
					log.Printf("Error getting player %d before creating game: %v", playerID, errPlayer)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Помилка отримання даних гравця."))
					break out
				}
				if inputData == ui.Yes { // Якщо "Так" - створюємо гру
					game := db.ProposedGame{ // Формуємо об'єкт гри
						PlayerID:      player.ID,
						RegionSection: singleGameChoice.Area,
						Partner:       singleGameChoice.Partner,
						Date:          singleGameChoice.Date, // Має бути timestamp
						Time:          singleGameChoice.Time,
						Court:         singleGameChoice.Court,
						Payment:       singleGameChoice.Payment,
					}
					errCreate := dbClient.CreateGame(game) // Зберігаємо в БД
					if errCreate != nil {
						log.Printf("Помилка створення гри в БД для гравця %d: %v", playerID, errCreate)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Помилка при збереженні гри..."))
					} else {
						log.Printf("Гра успішно створена в БД для гравця %d: %+v", playerID, game)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Вашу гру зареєстровано!"))
					}
				} else if inputData == ui.No { // Якщо "Ні" - скасовуємо
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Створення гри скасовано."))
				} else { // Якщо щось інше
					log.Printf("OneTimeGameHandler: Отримано неочікувані дані '%s' у стані AllSelected", inputData)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не зрозумів відповідь. Створення гри скасовано."))
				}
				log.Printf("OneTimeGameHandler: Завершення після AllSelected для %d.", playerID)
				ev_proc.mainMenu(chatID) // Повертаємось до головного меню
				break out                // Завершуємо рутину

			default: // Невідомий стан
				log.Printf("OneTimeGameHandler: Невідомий стан %v для користувача %d", state, playerID)
				if messageID != 0 {
					bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
				}
				ev_proc.mainMenu(chatID)
				break out
			} // end switch state
		} // end select
	} // end for
	log.Printf("OneTimeGameHandler: Вихід з функції для користувача %d.", playerID)
} // end func OneTimeGameHandler
