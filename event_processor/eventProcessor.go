// File: event_processor/eventProcessor.go
package eventprocessor

import (
	"fmt"
	"log"
	"regexp" // Залишаємо для майбутнього, якщо розкоментуємо parseMatchData
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	db "TennisBot/database"
	ui "TennisBot/ui"
)

// Constants (залишаємо)
const (
	StopProcessing  = "quit"
	PhotoFolderPath = "resources/avatarPhoto/" // Перевірте шлях! Можливо, має бути відносним або абсолютним.
)

// EventProcessor struct (залишаємо)
type EventProcessor struct {
	bot *tgbotapi.BotAPI
}

// Event struct (залишаємо)
type Event struct {
	ChatID int64
	Msg    string
}

// NewEventProcessor constructor (залишаємо)
func NewEventProcessor(bot *tgbotapi.BotAPI) EventProcessor {
	return EventProcessor{bot: bot}
}

// Функція зупинки рутини (перенесемо її сюди або в utils.go)
func stopRoutine(playerID int64, activeRoutines map[int64](chan string)) {
	if ch, exists := activeRoutines[playerID]; exists {
		log.Printf("Зупиняємо попередню рутину для %d", playerID)
		// Насилаємо команду виходу, якщо канал ще не закритий
		// Додаємо select, щоб не блокувати, якщо канал переповнений або закритий
		select {
		case ch <- ui.QuitChannelCommand:
		default:
				log.Printf("Не вдалося надіслати QuitChannelCommand для %d (канал закритий або переповнений?)", playerID)
		}
		// Закриваємо канал і видаляємо з мапи (defer в самій рутині також це зробить, але краще і тут)
		// Важливо: перевірити, чи це не викликає паніку при подвійному закритті
		// Краще покластися на defer в рутині, а тут просто видаляти з мапи?
		// Або додати перевірку в defer, як ми робили раніше.
		// Поки що залишимо так, але це потенційне місце для паніки "send on closed channel"
		// Якщо рутина вже завершилась, ch може бути nil після delete в її defer.
		// Безпечніше:
		// close(ch) // Може спричинити паніку, якщо вже закрито
		delete(activeRoutines, playerID) // Просто видаляємо з мапи, рутина завершиться сама
	}
}

// isMatchMessage перевіряє, чи містить повідомлення дані матчу.
func isMatchMessage(message string) bool {
	// Регулярний вираз для знаходження рахунку (наприклад, 6-3, 4-6)
	re := regexp.MustCompile(`\d{1,2}[-:]\d{1,2}(,\s*\d{1,2}[-:]\d{1,2})*`) // Можливо, варто уточнити
	match := re.FindString(message)
	return match != "" // Повертає true, якщо знайдено щось схоже на рахунок
}

// parseMatchData витягує імена гравців і рахунок з повідомлення.
func parseMatchData(message string) (playerA, playerB, score string, err error) {
	// TODO: Потрібен значно надійніший парсер!
	// Приклад простого парсингу: "PlayerA vs PlayerB 6-3, 4-6"
	parts := strings.Fields(message) // Розбиваємо за пробілами
	if len(parts) < 4 {
		return "", "", "", fmt.Errorf("недостатньо даних у повідомленні для парсингу матчу")
	}
	// Проста евристика: перший - гравець А, третій - гравець Б, решта - рахунок
	// Погано працюватиме з нікнеймами/іменами з кількох слів
	playerA = parts[0]
	if len(parts) > 2 && strings.ToLower(parts[1]) == "vs" {
		playerB = parts[2]
		if len(parts) > 3 {
			score = strings.Join(parts[3:], " ")
		} else {
			return "", "", "", fmt.Errorf("не знайдено рахунок у повідомленні")
		}
	} else {
		return "", "", "", fmt.Errorf("формат 'PlayerA vs PlayerB score...' не знайдено")
	}

	// Перевірка, чи знайдено рахунок за допомогою isMatchMessage
	if !isMatchMessage(score) {
		// return "", "", "", fmt.Errorf("не вдалося розпізнати рахунок '%s'", score)
		// Або ігноруємо помилку і повертаємо те, що знайшли
	}

	return playerA, playerB, score, nil
}

// processMatchResult обробляє результат матчу з вільного тексту.
// ПОТРЕБУЄ ЗНАЧНОЇ ДОРОБКИ для роботи з БД та визначення результату
func processMatchResult(playerAName, playerBName, score string, dbClient *db.DBClient) {
	log.Printf("Спроба обробки результату з тексту: %s vs %s, score: %s", playerAName, playerBName, score)

	// 1. Знайти гравців у БД за іменами/юзернеймами (дуже ненадійно!)
	// playerA, errA := dbClient.GetPlayerByUsername(playerAName) // Або за NameSurname?
	// playerB, errB := dbClient.GetPlayerByUsername(playerBName)
	// if errA != nil || errB != nil {
	//     log.Printf("Не вдалося знайти гравців '%s' або '%s' в БД", playerAName, playerBName)
	//     return
	// }

	// 2. Визначити результат (1.0 чи 0.0 для playerA) на основі рядка score (складно!)
	// resultA := determineWinnerFromResultString(score) // Потрібно реалізувати цю функцію

	// 3. Викликати оновлення рейтингу
	// errUpdate := ui.UpdatePlayerRating(playerA.UserID, playerB.UserID, resultA, dbClient)
	// if errUpdate != nil {
	//     log.Printf("Помилка оновлення рейтингу після парсингу тексту: %v", errUpdate)
	// } else {
	//     log.Printf("Рейтинг (можливо) оновлено після парсингу тексту для %s vs %s", playerAName, playerBName)
	// }
}

// Process обробляє всі вхідні події (повідомлення, колбеки).
func (ev_proc EventProcessor) Process(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), dbClient *db.DBClient) {

	// --- Обробка Повідомлень ---
	if update.Message != nil {
		playerID := update.Message.From.ID
		chatID := update.Message.Chat.ID
		isRegistered := dbClient.CheckPlayerRegistration(playerID)
		messageText := update.Message.Text

		// 1. Обробка команд ПЕРЕД УСІМ
		if update.Message.IsCommand() {
			stopRoutine(playerID, activeRoutines) // Зупиняємо попередню дію
			ev_proc.processCommand(bot, update.Message.Command(), update, activeRoutines, dbClient)
			return
		}

		// 2. Обробка Контакту/Фото (вони зазвичай в рамках якоїсь рутини)
		if update.Message.Contact != nil {
			if ch, exists := activeRoutines[playerID]; exists {
				ch <- update.Message.Contact.PhoneNumber + ":" + update.Message.From.UserName
			} // Не зупиняємо рутину, бо це частина процесу
			return
		}
		// oбробка Фото
		if len(update.Message.Photo) > 0 {
			if ch, exists := activeRoutines[playerID]; exists {
				ch <- update.Message.Photo[len(update.Message.Photo)-1].FileID
			} // Не зупиняємо рутину
			return
		}

		// Якщо активної рутини немає, обробляємо текст як команду меню або інше
		// 3. Обробка кнопок головного меню (з перериванням попередньої дії)
		switch messageText {
		case ui.ProfileButton:
			stopRoutine(playerID, activeRoutines) // Зупиняємо
			if isRegistered {
				ev_proc.ProfileButtonHandler(bot, chatID, playerID, dbClient)
			} else {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)
			}
			return // Дія виконана
		case ui.SingleGame:
			stopRoutine(playerID, activeRoutines) // Зупиняємо
			if isRegistered {
				go ev_proc.OneTimeGameHandler(bot, update, activeRoutines, playerID, dbClient)
			} else {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)
			}
			return // Дія виконана (запуск горутини)
		case ui.GeneralRatingButton:
			stopRoutine(playerID, activeRoutines) // Зупиняємо
			if isRegistered {
				rating := ui.GetPlayerRating(playerID, dbClient)
				msg := tgbotapi.NewMessage(chatID, rating)
				bot.Send(msg)
			} else {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)
			}
			return // Дія виконана
		case ui.FixScoreButton:
			stopRoutine(playerID, activeRoutines) // Зупиняємо
			if isRegistered {
				ev_proc.StartFixScoreFlow(bot, chatID, playerID, dbClient, activeRoutines)
			} else {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)
			}
			return // Дія виконана (запуск горутини)
		case ui.TournamentsButton:
			stopRoutine(playerID, activeRoutines) // Зупиняємо
			if isRegistered {
				ev_proc.TournamentsButtonHandler(bot, chatID, playerID, dbClient)
			} else {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)
			}
			return // Дія виконана
		case ui.HelpButton:
			stopRoutine(playerID, activeRoutines) // Зупиняємо
			if isRegistered {
				ev_proc.HelpButtonHandler(bot, chatID, playerID, dbClient)
			} else {
				ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient)
			}
			return // Дія виконана

		// 4. Якщо це НЕ команда/кнопка меню - передаємо текст в активну рутину, ЯКЩО вона є
		if ch, exists := activeRoutines[playerID]; exists {
			log.Printf("Передаємо текст '%s' в активну рутину для %d", messageText, playerID)
			// Використовуємо неблокуюче надсилання, щоб уникнути зависання, якщо рутина не читає
			select {
			case ch <- messageText:
				// Успішно надіслано
			default:
				log.Printf("Помилка передачі тексту '%s' в рутину для %d: канал заблоковано або закрито.", messageText, playerID)
				// Можливо, рутина зависла або завершилась некоректно
				// Можна спробувати її зупинити?
				stopRoutine(playerID, activeRoutines)
			}
		} else {
			// Якщо активної рутини немає і це не кнопка меню - ігноруємо або відповідаємо
			log.Printf("Повідомлення '%s' від %d не оброблено (немає активної рутини/не кнопка меню).", messageText, playerID)
			// ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не зрозумів команду. Скористайтеся кнопками або /menu."))
		}
		default:
			// Якщо немає активної рутини і текст не кнопка меню - ігноруємо або відповідаємо стандартно
			log.Printf("Повідомлення '%s' від %d не оброблено (немає активної рутини/не кнопка меню).", messageText, playerID)
			// Можна надіслати підказку
			// ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не зрозумів команду. Скористайтеся кнопками або /menu."))

		}
		// --- Кінець обробки повідомлень ---

	} else if update.CallbackQuery != nil {
		// --- Обробка Натискань Кнопок (CallbackQuery) ---
		playerID := update.CallbackQuery.From.ID
		chatID := update.CallbackQuery.Message.Chat.ID
		callbackData := update.CallbackQuery.Data
		messageID := update.CallbackQuery.Message.MessageID // ID повідомлення з кнопками

		log.Printf("Отримано callback: '%s' від %d", callbackData, playerID)

		currentUser, errUser := dbClient.GetPlayer(playerID)
		
		// Працюємо з зареєстрованими користувачами (для більшості колбеків)
		// isRegistered := dbClient.CheckPlayerRegistration(playerID)
		// TODO: Додати перевірку isRegistered для відповідних колбеків, якщо потрібно

		// Видаляємо годинник очікування у користувача
		callbackResp := tgbotapi.NewCallback(update.CallbackQuery.ID, "") // Пустий текст відповіді

		// Обробляємо специфічні колбеки тут
		switch {
		// --- Колбеки фіксації рахунку ---
		case strings.HasPrefix(callbackData, "fix_score_result:"): // "fix_score_result:opponentID:result"
			parts := strings.Split(callbackData, ":")
			if len(parts) != 3 {
				log.Printf("Помилка: Невірний формат callback '%s'", callbackData)
				callbackResp.Text = "Помилка даних"
				bot.Request(callbackResp)
				return
			}
			opponentID_int64, errOp := strconv.ParseInt(parts[1], 10, 64)
			result_float64, errRes := strconv.ParseFloat(parts[2], 64) // 1.0 або 0.0

			if errOp != nil || errRes != nil || (result_float64 != 1.0 && result_float64 != 0.0) {
				log.Printf("Помилка парсингу callback '%s': %v, %v", callbackData, errOp, errRes)
				callbackResp.Text = "Помилка обробки результату"
				bot.Request(callbackResp)
				return
			}

			// Видаляємо кнопки "Я виграв/програв"
			editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
			bot.Request(editMsg)

			// Оновлюємо рейтинг
			errUpdate := ui.UpdatePlayerRating(playerID, opponentID_int64, result_float64, dbClient)
			if errUpdate != nil {
				log.Printf("Помилка оновлення рейтингу (callback) для %d vs %d: %v", playerID, opponentID_int64, errUpdate)
				bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Помилка при оновленні рейтингу: %v", errUpdate)))
				callbackResp.Text = "Помилка оновлення рейтингу"
			} else {
				log.Printf("Рейтинг оновлено (callback) для %d vs %d, результат A=%.1f", playerID, opponentID_int64, result_float64)
				newRatingMsg := ui.GetPlayerRating(playerID, dbClient) // Отримуємо оновлений рейтинг
				bot.Send(tgbotapi.NewMessage(chatID, "Рахунок зафіксовано!\n"+newRatingMsg))
				callbackResp.Text = "Рахунок зафіксовано!" // Відповідь на колбек
			}
			bot.Request(callbackResp)
			// Зупиняємо відповідну рутину фіксації рахунку, якщо вона ще існує
			stopRoutine(playerID, activeRoutines) // Зупиняємо рутину handleFixScoreRoutine

		case callbackData == "cancel_fix_score":
			// Видаляємо кнопки "Я виграв/програв"
			editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
			bot.Request(editMsg)
			bot.Send(tgbotapi.NewMessage(chatID, "Фіксацію рахунку скасовано."))
			callbackResp.Text = "Скасовано"
			bot.Request(callbackResp)
			// Зупиняємо рутину фіксації
			stopRoutine(playerID, activeRoutines)

		// --- Колбеки підтвердження гри (відгук на пропозицію) ---
		case strings.HasPrefix(callbackData, "confirm_game:yes:"): // "confirm_game:yes:gameID" або "confirm_game:no:gameID"
				parts := strings.Split(callbackData, ":")
				if len(parts) != 3 {
						log.Printf("Помилка: Невірний формат callback '%s'", callbackData)
						callbackResp.Text = "Помилка даних"
						bot.Request(callbackResp)
						return
				}
				confirmation := parts[1] // "yes" або "no"
				gameID_uint64, errGameID := strconv.ParseUint(parts[2], 10, 64)
				if errGameID != nil {
						log.Printf("Помилка парсингу gameID з callback '%s': %v", callbackData, errGameID)
						callbackResp.Text = "Помилка ID гри"
						bot.Request(callbackResp)
						return
				}
				gameID := uint(gameID_uint64)

				// Видаляємо кнопки Yes/No у поточного користувача
				editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
				bot.Request(editMsg)

				if confirmation == "yes" {
						log.Printf("Гравець %d підтвердив гру %d", playerID, gameID)
						// 1. Отримати гру та ID гравця, що її запропонував
						game, errGame := dbClient.GetGame(gameID)
						if errGame != nil {
								log.Printf("Помилка отримання гри %d при підтвердженні: %v", gameID, errGame)
								bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося знайти гру для підтвердження."))
								callbackResp.Text = "Гра не знайдена"
								bot.Request(callbackResp)
								return
						}
						proposerID := game.UserID

						// 2. Отримати дані гравця, що ПІДТВЕРДИВ (поточного playerID)
						responder, errResponder := dbClient.GetPlayer(playerID)
						if errResponder != nil {
								log.Printf("Помилка отримання даних гравця %d, що підтвердив гру %d: %v", playerID, gameID, errResponder)
								// Продовжуємо, але не зможемо надіслати дані
						}

						// 3. Надіслати повідомлення гравцю, що запропонував гру
						msgToProposerText := fmt.Sprintf("Гравець %s (@%s) відгукнувся на вашу пропозицію:\n%s\n\nЗв'яжіться для узгодження деталей.",
								responder.NameSurname, // Ім'я того, хто відгукнувся
								strings.TrimPrefix(responder.UserName, "@"),
								game.String(), // Деталі гри
						)
						msgToProposer := tgbotapi.NewMessage(proposerID, msgToProposerText)
						_, errSend := bot.Send(msgToProposer)
						if errSend != nil {
								log.Printf("Помилка надсилання повідомлення про підтвердження гри %d гравцю %d: %v", gameID, proposerID, errSend)
								// Повідомити поточного користувача, що не вдалося сповістити?
						} else {
								// Повідомляємо поточного користувача, що його згоду надіслано
								bot.Send(tgbotapi.NewMessage(chatID, "Вашу згоду надіслано гравцю! Очікуйте на повідомлення від нього або зв'яжіться першим."))
						}

						// 4. Можливо, створити запис у DualGame або змінити статус ProposedGame?
						// dbClient.DB.Create(&db.DualGame{ProposedPlayerID: proposerID, RespondedPlayerID: playerID, ConfirmationProposed: true, ConfirmationResponded: true, BothConfirmed: false})

						// 5. Видалити ProposedGame, щоб вона більше не відображалася
						errDelete := dbClient.DeleteGame(gameID)
						if errDelete != nil {
								log.Printf("Помилка видалення ProposedGame %d після підтвердження: %v", gameID, errDelete)
						}

						callbackResp.Text = "Згоду надіслано"

				} else { // confirmation == "no"
						log.Printf("Гравець %d відхилив гру %d", playerID, gameID)
						callbackResp.Text = "Гру відхилено"
						bot.Send(tgbotapi.NewMessage(chatID, "Ви відхилили пропозицію гри."))
						// Можна сповістити гравця, що запропонував (опціонально)
				}
				bot.Request(callbackResp) // Відповідаємо на колбек

		// --- Інші колбеки (редагування профілю, видалення ігор) ---
		case callbackData == ui.EditOptionPhoto:	
			stopRoutine(playerID, activeRoutines) // Зупинка попередньої рутини, якщо є
			// Запускаємо відповідний обробник, передаючи dbClient
			go ev_proc.ProfilePhotoEditButtonHandler(bot, update, activeRoutines, playerID, dbClient)
		case callbackData == ui.EditOptionRacket:
			stopRoutine(playerID, activeRoutines)
			// Запускаємо відповідний обробник, передаючи dbClient
			go ev_proc.ProfileRacketEditButtonHandler(bot, update, activeRoutines, playerID, dbClient)
		case callbackData == ui.DeleteGames:
			stopRoutine(playerID, activeRoutines)
			// Запускаємо відповідний обробник, передаючи dbClient
			go ev_proc.DeleteGames(bot, update, activeRoutines, playerID, dbClient)

		// --- Обробка колбеків, які мають йти в активні рутини ---
		default: // Якщо колбек не оброблений явно
                if ch, exists := activeRoutines[playerID]; exists {
                    log.Printf("Передаємо callback '%s' в активну рутину для %d", callbackData, playerID)
                    select {
                    case ch <- callbackData:
                         // Успішно. Відповісти на колбек тут чи в рутині?
                         // Якщо відповісти тут, то користувач одразу побачить реакцію.
                         // bot.Request(callbackResp)
                    default:
                        log.Printf("Помилка передачі callback '%s' в рутину для %d: канал заблоковано або закрито.", callbackData, playerID)
                        // stopRoutine(playerID, activeRoutines) // Можливо, треба зупинити завислу рутину
                    }
                } else {
                    log.Printf("Невідомий callback '%s' від %d (немає активної рутини)", callbackData, playerID)
                    callbackResp.Text = "Невідома дія"
                    bot.Request(callbackResp)
                }
		} // end switch callback

	} // end if update.CallbackQuery != nil
} // end func Process
// я оновила папку