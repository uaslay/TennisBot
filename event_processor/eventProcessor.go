// File: event_processor/eventProcessor.go
package eventprocessor

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"errors" // Додаємо імпорт errors
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm" // Додаємо імпорт gorm

	db "TennisBot/database"
	ui "TennisBot/ui"
)

// Constants
const (
	StopProcessing = "quit" // Змінено для відповідності ui.QuitChannelCommand
	// PhotoFolderPath = "resources/avatarPhoto/" // Закоментовано, бо переходимо на FileID
)

// EventProcessor struct
type EventProcessor struct {
	bot *tgbotapi.BotAPI
}

// Event struct
type Event struct {
	ChatID int64
	Msg    string
}

// NewEventProcessor constructor
func NewEventProcessor(bot *tgbotapi.BotAPI) EventProcessor {
	return EventProcessor{bot: bot}
}

// Функція зупинки рутини
func stopRoutine(playerID int64, activeRoutines map[int64](chan string)) {
	if ch, exists := activeRoutines[playerID]; exists {
		log.Printf("Зупиняємо попередню рутину для %d", playerID)
		delete(activeRoutines, playerID)
		close(ch)
		log.Printf("Канал для рутини користувача %d закрито.", playerID)
	} else {
		log.Printf("Немає активної рутини для зупинки для користувача %d.", playerID)
	}
}

// isMatchMessage (залишаємо без змін)
func isMatchMessage(message string) bool {
	re := regexp.MustCompile(`\d{1,2}[-:]\d{1,2}(,\s*\d{1,2}[-:]\d{1,2})*`)
	match := re.FindString(message)
	return match != ""
}

// parseMatchData (залишаємо без змін)
func parseMatchData(message string) (playerA, playerB, score string, err error) {
	parts := strings.Fields(message)
	if len(parts) < 4 {
		return "", "", "", fmt.Errorf("недостатньо даних у повідомленні для парсингу матчу")
	}
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
	if !isMatchMessage(score) {
		// Можна ігнорувати або повертати помилку
	}
	return playerA, playerB, score, nil
}

// processMatchResult (залишаємо без змін, але потребує реалізації)
func processMatchResult(playerAName, playerBName, score string, dbClient *db.DBClient) {
	log.Printf("Спроба обробки результату з тексту: %s vs %s, score: %s", playerAName, playerBName, score)
	// TODO: Реалізувати логіку
}

// Helper function to send message safely
func (ev_proc *EventProcessor) sendMessage(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
	return ev_proc.bot.Send(msg)
}

// Helper function to request (edit, delete, answerCallback) safely
func (ev_proc *EventProcessor) request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return ev_proc.bot.Request(c)
}


// Process обробляє всі вхідні події (повідомлення, колбеки).// Process обробляє всі вхідні події (повідомлення, колбеки).
func (ev_proc EventProcessor) Process(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), dbClient *db.DBClient) {

	var playerID int64
	var chatID int64
	var isCallback bool
	var callbackQueryID string
	var messageID int
	var messageText string // Додамо змінну для тексту повідомлення
	var dataToSend string  // Дані для передачі в рутину
	var dataType string    // Тип даних для передачі

	// --- Визначення основних даних з update ---
	if update.Message != nil {
		playerID = update.Message.From.ID
		chatID = update.Message.Chat.ID
		isCallback = false
		messageText = update.Message.Text // Зберігаємо текст одразу

		// Визначаємо дані для передачі в рутину (якщо вона є)
		if len(update.Message.Photo) > 0 {
			dataToSend = update.Message.Photo[len(update.Message.Photo)-1].FileID
			dataType = "photo"
		} else if update.Message.Contact != nil {
			username := update.Message.From.UserName
			if username == "" {
				username = "unknown" // Або інша заглушка
			}
			dataToSend = update.Message.Contact.PhoneNumber + ":" + username
			dataType = "contact"
		} else if messageText != "" {
			dataToSend = messageText
			dataType = "message"
		} // Якщо ні текст, ні фото, ні контакт - dataToSend залишиться порожнім

	} else if update.CallbackQuery != nil {
		playerID = update.CallbackQuery.From.ID
		chatID = update.CallbackQuery.Message.Chat.ID
		isCallback = true
		callbackQueryID = update.CallbackQuery.ID
		messageID = update.CallbackQuery.Message.MessageID
		// Для колбеків дані для передачі - це самі дані колбеку
		dataToSend = update.CallbackQuery.Data
		dataType = "callback"
	} else {
		log.Println("Process: Невідомий тип update")
		return // Не можемо обробити
	}

	// --- Відповідь на Callback Query (якщо це колбек) ---
	// Робимо це одразу, щоб уникнути "зависання" кнопки
	if isCallback {
		callbackResp := tgbotapi.NewCallback(callbackQueryID, "") // Порожня відповідь за замовчуванням
		_, err := ev_proc.request(callbackResp)
		if err != nil {
			log.Printf("Помилка відповіді на callback query %s: %v", callbackQueryID, err)
			// Не критично, продовжуємо обробку
		}
	}

	// === НОВА ЛОГІКА: Спочатку перевіряємо, чи є активна рутина ===
	if ch, routineExists := activeRoutines[playerID]; routineExists {
		log.Printf("Process: Активна рутина існує для %d. Передаємо дані '%s' (%s).", playerID, dataToSend, dataType)
		if dataToSend != "" { // Перевіряємо, чи є що передавати
			select {
			case ch <- dataToSend:
				// Дані успішно передано в активну рутину (реєстрації, гри, редагування тощо)
			default:
				// Канал заблоковано (рутина "зависла" або не встигає обробити?)
				// Можливо, варто зупинити рутину в такому випадку
				log.Printf("ПОМИЛКА: Не вдалося передати дані '%s' (%s) в рутину для %d: канал заблоковано. Зупиняємо рутину.", dataToSend, dataType, playerID)
				stopRoutine(playerID, activeRoutines) // Зупиняємо проблемну рутину
				ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Сталася помилка при обробці попередньої дії. Спробуйте ще раз."))
			}
		} else if !isCallback { // Якщо це не колбек і немає даних (дивне повідомлення?)
			log.Printf("Process: Отримано повідомлення без тексту/фото/контакту від %d з активною рутиною. Ігноруємо.", playerID)
		}
		// Після передачі даних (або ігнорування) в активну рутину, завершуємо обробку цього update
		return
	}

	// === Якщо активної рутини НЕМАЄ, продовжуємо стандартну логіку ===
	log.Printf("Process: Немає активної рутини для %d. Обробляємо як нову дію.", playerID)

	// --- Перевірка Реєстрації (тільки якщо рутини немає) ---
	isRegistered := dbClient.CheckPlayerRegistration(playerID)

	// --- Логіка для НЕЗАРЕЄСТРОВАНИХ користувачів (і без активної рутини) ---
	if !isRegistered {
		if !isCallback && !update.Message.IsCommand() && messageText != ui.StartCommand {
			// Якщо це звичайне повідомлення (не колбек, не команда, не /start) від незареєстрованого користувача
			// І ми дійшли сюди (тобто рутини немає) - ЗАПУСКАЄМО РЕЄСТРАЦІЮ
			log.Printf("User %d is not registered and no routine active. Initiating registration flow.", playerID)
			// stopRoutine(playerID, activeRoutines) // Не потрібно, бо рутини немає (перевірили вище)
			ev_proc.registrationFlowHandler(bot, update, activeRoutines, dbClient) // Запускаємо реєстрацію
			return                                                                   // Завершуємо обробку цього update
		} else if isCallback {
			// Колбек від незареєстрованого (і без рутини) - ігноруємо, просимо зареєструватись
			log.Printf("Ignoring callback from unregistered user %d without active routine.", playerID)
			ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Будь ласка, спочатку зареєструйтесь або увійдіть: /start"))
			return
		}
		// Якщо це команда /start від незареєстрованого, вона буде оброблена нижче як команда
		// Якщо це інша команда - теж буде оброблена нижче (але processCommand перевірить реєстрацію знову)
	}

	// --- Обробка Дій для ЗАРЕЄСТРОВАНИХ або Команд (/start) для всіх (і без активної рутини) ---

	var highLevelActionTriggered bool = false
	var actionFunc func()

	if !isCallback { // Обробляємо лише повідомлення як високорівневі тригери
		// 1. Команди
		if update.Message.IsCommand() {
			highLevelActionTriggered = true
			command := update.Message.Command()
			actionFunc = func() {
				// Передаємо isRegistered, щоб уникнути повторної перевірки в processCommand
				ev_proc.processCommand(bot, command, update, activeRoutines, dbClient, isRegistered)
			}
		} else {
			// 2. Кнопки головного меню (тільки для зареєстрованих)
			if isRegistered { // Додаємо перевірку!
				switch messageText {
				case ui.ProfileButton:
					highLevelActionTriggered = true
					actionFunc = func() {
						ev_proc.ProfileButtonHandler(bot, chatID, playerID, dbClient)
					}
				case ui.SingleGame:
					highLevelActionTriggered = true
					actionFunc = func() {
						// Запускаємо гру (вона сама створить рутину)
						// Важливо: Передаємо update, щоб OneTimeGameHandler міг отримати messageID і т.д.
						go ev_proc.OneTimeGameHandler(bot, update, activeRoutines, playerID, dbClient)
					}
				case ui.GeneralRatingButton:
					highLevelActionTriggered = true
					actionFunc = func() {
						rating := ui.GetPlayerRating(playerID, dbClient)
						msg := tgbotapi.NewMessage(chatID, rating)
						ev_proc.sendMessage(msg)
						ev_proc.mainMenu(chatID) // Показуємо меню після рейтингу
					}
				case ui.FixScoreButton:
					highLevelActionTriggered = true
					actionFunc = func() {
						// Запускаємо процес фіксації (він створить рутину)
						ev_proc.StartFixScoreFlow(bot, chatID, playerID, dbClient, activeRoutines)
					}
				// TODO: Додати обробку інших кнопок меню, якщо вони є ("Турніри", "Допомога")
				default:
					// Якщо текст не команда і не кнопка меню
					log.Printf("Process: Невідоме повідомлення '%s' від зареєстрованого користувача %d без активної рутини.", messageText, playerID)
					// Можливо, просто показати головне меню?
					// ev_proc.mainMenu(chatID)
				}
			} // end if isRegistered
		}
	} else { // Це CallbackQuery (і рутини немає)
		// Обробка колбеків БЕЗ Активної Рутини
		callbackData := dataToSend // Ми вже зберегли update.CallbackQuery.Data в dataToSend

		log.Printf("Process: Обробка callback '%s' від %d (ID повідомлення: %d) без активної рутини.", callbackData, playerID, messageID)

		parts := strings.Split(callbackData, ":")
		command := parts[0]

		// Обробляємо тільки якщо користувач зареєстрований АБО це специфічний колбек (якщо такі є)
		if isRegistered { // Додаємо перевірку!
			switch command {
			case "confirm_game":
                highLevelActionTriggered = true
                actionFunc = func() {
                    // Перевіряємо формат колбеку
                    if len(parts) != 3 {
                        log.Printf("Помилка: Невірний формат callback '%s'", callbackData)
                        ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Помилка обробки відповіді на гру."))
                        return
                    }
                    confirmation := parts[1]
                    // Конвертуємо ID гри
                    gameID_uint64, errGameID := strconv.ParseUint(parts[2], 10, 64)

                    // ---> ДОДАЄМО ПЕРЕВІРКУ НА ПОМИЛКУ ТУТ <---
                    if errGameID != nil {
                        log.Printf("Помилка парсингу gameID з callback '%s': %v", callbackData, errGameID)
                        ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Помилка обробки ID гри."))
                        return // Виходимо, якщо ID не розпарсився
                    }
                    // ---> Кінець перевірки <---

                    // Якщо помилки не було, продовжуємо
                    editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
                    ev_proc.request(editMsg) // Прибираємо кнопки після натискання

                    if confirmation == "yes" {
                        ev_proc.handleGameResponseYes(chatID, playerID, uint(gameID_uint64), dbClient)
                    } else {
                        ev_proc.handleGameResponseNo(chatID, playerID, uint(gameID_uint64), dbClient)
                    }
                }

			case "manage_responses":
                highLevelActionTriggered = true
                actionFunc = func() {
                    // Перевіряємо, чи правильна кількість частин у колбеку
                    if len(parts) != 2 {
                        log.Printf("Помилка: Невірний формат callback '%s'", callbackData)
                        ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Помилка керування відгуками."))
                        return // Виходимо, якщо формат неправильний
                    }
                    // Конвертуємо ID гри
                    gameID_uint64, errGameID := strconv.ParseUint(parts[1], 10, 64)
                    // ---> Ось ТУТ потрібна перевірка <---
                    if errGameID != nil {
                        log.Printf("Помилка парсингу gameID з callback '%s': %v", callbackData, errGameID)
                        ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Помилка обробки ID гри."))
                        return // Виходимо, якщо ID не розпарсився
                    }
                    // Якщо помилки не було, продовжуємо
                    ev_proc.request(tgbotapi.NewDeleteMessage(chatID, messageID))
                    ev_proc.handleManageResponses(chatID, playerID, uint(gameID_uint64), dbClient)
                }

			case "choose_opponent":
				highLevelActionTriggered = true
				actionFunc = func() {
					// ... (логіка choose_opponent) ...
					// Потрібно перевірити len(parts) == 3
					gameID_uint64, errGameID := strconv.ParseUint(parts[1], 10, 64)
					responderID_int64, errResponderID := strconv.ParseInt(parts[2], 10, 64)
					if errGameID != nil || errResponderID != nil || len(parts) != 3 {
						// обробка помилки
						return
					}
					ev_proc.request(tgbotapi.NewDeleteMessage(chatID, messageID))
					ev_proc.handleChooseOpponent(chatID, playerID, uint(gameID_uint64), responderID_int64, dbClient)
				}

			case "cancel_proposal":
				highLevelActionTriggered = true
				actionFunc = func() {
					// ... (логіка cancel_proposal) ...
					// Потрібно перевірити len(parts) == 2
					gameID_uint64, errGameID := strconv.ParseUint(parts[1], 10, 64)
					if errGameID != nil || len(parts) != 2 {
						// обробка помилки
						return
					}
					ev_proc.request(tgbotapi.NewDeleteMessage(chatID, messageID))
					ev_proc.handleCancelProposal(chatID, playerID, uint(gameID_uint64), dbClient)
				}

			case ui.EditOptionPhoto:
				highLevelActionTriggered = true
				actionFunc = func() {
					// Запускаємо рутину редагування фото
					go ev_proc.ProfilePhotoEditButtonHandler(bot, update, activeRoutines, playerID, dbClient)
				}
			case ui.EditOptionRacket:
				highLevelActionTriggered = true
				actionFunc = func() {
					// Запускаємо рутину редагування ракетки
					go ev_proc.ProfileRacketEditButtonHandler(bot, update, activeRoutines, playerID, dbClient)
				}
			case ui.DeleteGames:
				highLevelActionTriggered = true
				actionFunc = func() {
					// Запускаємо рутину видалення ігор
					go ev_proc.DeleteGames(bot, update, activeRoutines, playerID, dbClient)
				}

			case "fix_score_result":
				highLevelActionTriggered = true
				actionFunc = func() {
					// ... (логіка fix_score_result) ...
					// Потрібно перевірити len(parts) == 3
					opponentID_int64, errOp := strconv.ParseInt(parts[1], 10, 64)
					result_float64, errRes := strconv.ParseFloat(parts[2], 64)
					if errOp == nil && errRes == nil && len(parts) == 3 && (result_float64 == 1.0 || result_float64 == 0.0) {
						log.Printf("Обробка callback 'fix_score_result' для %d vs %d", playerID, opponentID_int64)
						editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
						ev_proc.request(editMsg)
						errUpdate := ui.UpdatePlayerRating(playerID, opponentID_int64, result_float64, dbClient)
						if errUpdate != nil {
							ev_proc.sendMessage(tgbotapi.NewMessage(chatID, fmt.Sprintf("Помилка при оновленні рейтингу: %v", errUpdate)))
						} else {
							newRatingMsg := ui.GetPlayerRating(playerID, dbClient)
							ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Рахунок зафіксовано!\n"+newRatingMsg))
						}
						// Можливо, показати головне меню після фіксації
						ev_proc.mainMenu(chatID)
					} else {
						log.Printf("Помилка парсингу або невірний формат callback 'fix_score_result': %s", callbackData)
					}
				}

			case "cancel_fix_score":
				highLevelActionTriggered = true
				actionFunc = func() {
					// ... (логіка cancel_fix_score) ...
					log.Printf("Обробка callback 'cancel_fix_score' для %d", playerID)
					editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
					ev_proc.request(editMsg)
					ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Фіксацію рахунку скасовано."))
					ev_proc.mainMenu(chatID)
				}

			case ui.MyProposedGamesCallback:
				highLevelActionTriggered = true
				actionFunc = func() {
					log.Printf("Обробка callback '%s' від %d", callbackData, playerID)
					ev_proc.request(tgbotapi.NewDeleteMessage(chatID, messageID)) // Видаляємо повідомлення зі списком ігор
					ev_proc.MyProposedGamesHandler(bot, chatID, playerID, dbClient)
				}

			case "main_menu_from_my_games":
				highLevelActionTriggered = true
				actionFunc = func() {
					log.Printf("Обробка callback '%s' від %d", callbackData, playerID)
					ev_proc.request(tgbotapi.NewDeleteMessage(chatID, messageID)) // Видаляємо повідомлення зі списком "Мої ігри"
					ev_proc.mainMenu(chatID)
				}

			default:
				log.Printf("Process: Невідомий або необроблений callback '%s' від зареєстрованого користувача %d (немає активної рутини)", callbackData, playerID)
				// Можливо, показати головне меню?
				// ev_proc.mainMenu(chatID)
			} // end switch command (callback)
		} else { // Колбек від НЕзареєстрованого користувача
			log.Printf("Ignoring callback '%s' from unregistered user %d without active routine.", callbackData, playerID)
			ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Будь ласка, спочатку зареєструйтесь або увійдіть: /start"))
			return
		}
	} // end if isCallback

	// --- Виконання високорівневої дії (якщо вона була визначена) ---
	if highLevelActionTriggered && actionFunc != nil {
		log.Printf("Process: Виконуємо високорівневу дію для %d.", playerID)
		// stopRoutine(playerID, activeRoutines) // Зупинка не потрібна, бо ми сюди дійшли тільки якщо рутини не було
		actionFunc() // Виконуємо дію (деякі з них запускають нові рутини)
		return       // Завершуємо обробку
	}

	// Якщо ми дійшли сюди, це означає, що:
	// 1. Не було активної рутини.
	// 2. Це не повідомлення/колбек, який запускає реєстрацію.
	// 3. Це не повідомлення/колбек, який є відомою командою/кнопкою/коллбеком без рутини.
	// Отже, це якась неочікувана дія.
	log.Printf("Process: Дійшли до кінця функції без обробки для playerID %d (isCallback: %t, isRegistered: %t, messageText: '%s', callbackData: '%s')",
		playerID, isCallback, isRegistered, messageText, dataToSend)
	// Можливо, варто показати головне меню як реакцію за замовчуванням?
	// if isRegistered {
	// 	ev_proc.mainMenu(chatID)
	// }

} // end func Process


// --- Handlers for Game Response Logic ---

func (ev_proc *EventProcessor) handleGameResponseYes(chatID, responderID int64, gameID uint, dbClient *db.DBClient) {
	log.Printf("handleGameResponseYes: Гравець %d відгукнувся на гру %d", responderID, gameID)

	game, errGame := dbClient.GetGame(gameID)
	if errGame != nil {
		log.Printf("handleGameResponseYes: Помилка отримання гри %d: %v", gameID, errGame)
		msgText := "Помилка: Не вдалося знайти гру."
		if errors.Is(errGame, gorm.ErrRecordNotFound) {
			msgText = "На жаль, ця гра вже неактуальна або була видалена."
		}
		ev_proc.sendMessage(tgbotapi.NewMessage(chatID, msgText))
		return
	}

	proposerID := game.Player.UserID
	if responderID == proposerID {
		log.Printf("handleGameResponseYes: Гравець %d намагається відгукнутися на власну гру %d", responderID, gameID)
		ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Ви не можете відгукнутися на власну пропозицію гри."))
		return
	}

	alreadyResponded, errCheck := dbClient.CheckExistingResponse(gameID, responderID)
	if errCheck != nil {
		log.Printf("handleGameResponseYes: Помилка перевірки існуючого відгуку для гри %d, гравця %d: %v", gameID, responderID, errCheck)
		ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Сталася помилка при обробці вашого відгуку. Спробуйте пізніше."))
		return
	}
	if alreadyResponded {
		log.Printf("handleGameResponseYes: Гравець %d вже відгукувався на гру %d", responderID, gameID)
		ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Ви вже відгукувалися на цю гру."))
		return
	}

	responderPlayer, errResponder := dbClient.GetPlayer(responderID)
	if errResponder != nil {
		// Обробка помилки: не вдалося знайти гравця-відповідача
		log.Printf("handleGameResponseYes: Error getting responder player %d: %v", responderID, errResponder)
		ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Сталася помилка при отриманні даних гравця."))
		return // Або інша логіка обробки помилки
	}

	gameResponse := db.GameResponse{ProposedGameID: gameID, ResponderID: responderPlayer.ID}
	errCreate := dbClient.CreateGameResponse(gameResponse)
	if errCreate != nil {
		log.Printf("handleGameResponseYes: Помилка створення GameResponse для гри %d, гравця %d: %v", gameID, responderID, errCreate)
		ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Сталася помилка при збереженні вашого відгуку."))
		return
	}

	// === ВИПРАВЛЕННЯ: proposer оголошено тут ===
	// Сповіщення пропозера
	_, errProposer := dbClient.GetPlayer(proposerID)
	if errProposer != nil {
		log.Printf("handleGameResponseYes: Помилка отримання пропозера %d для сповіщення: %v", proposerID, errProposer)
	} else {
		proposerChatID := proposerID
		responseText := fmt.Sprintf("🔔 Новий відгук на вашу гру!\n\nГра: %s\n\nГравець: %s (@%s, Рейтинг: %.0f)",
			game.String(), responderPlayer.NameSurname, strings.TrimPrefix(responderPlayer.UserName, "@"), responderPlayer.Rating) // Використовуємо responderPlayer
		manageCallbackData := fmt.Sprintf("manage_responses:%d", gameID)
		manageKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🧐 Керувати відгуками", manageCallbackData),
			),
		)
		msgToProposer := tgbotapi.NewMessage(proposerChatID, responseText)
		msgToProposer.ReplyMarkup = manageKeyboard
		_, errSend := ev_proc.sendMessage(msgToProposer)
		if errSend != nil {
			log.Printf("handleGameResponseYes: Помилка надсилання сповіщення пропозеру %d: %v", proposerID, errSend)
		} else {
			log.Printf("handleGameResponseYes: Сповіщення про новий відгук надіслано пропозеру %d", proposerID)
		}
	}
	// =========================================

	ev_proc.sendMessage(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Ваш відгук на гру '%s' надіслано гравцю %s. Очікуйте на підтвердження.", game.String(), game.Player.NameSurname)))
}

func (ev_proc *EventProcessor) handleGameResponseNo(chatID, responderID int64, gameID uint, dbClient *db.DBClient) {
	log.Printf("handleGameResponseNo: Гравець %d відхилив гру %d", responderID, gameID)
	ev_proc.sendMessage(tgbotapi.NewMessage(chatID, "Ви відхилили пропозицію гри."))
}

func (ev_proc *EventProcessor) handleManageResponses(proposerChatID, proposerID int64, gameID uint, dbClient *db.DBClient) {
	log.Printf("handleManageResponses: Пропозер %d керує відгуками на гру %d", proposerID, gameID)

	game, errGame := dbClient.GetGame(gameID)
	if errGame != nil {
		log.Printf("handleManageResponses: Помилка отримання гри %d: %v", gameID, errGame)
		msgText := "Помилка: Не вдалося знайти гру."
		if errors.Is(errGame, gorm.ErrRecordNotFound) {
			msgText = "На жаль, ця гра вже неактуальна або була видалена."
		}
		ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, msgText))
		return
	}
	if game.Player.UserID != proposerID {
		log.Printf("handleManageResponses: Гравець %d намагається керувати відгуками на чужу гру %d", proposerID, gameID)
		ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, "Ви не можете керувати відгуками на цю гру."))
		return
	}

	responses, errResponses := dbClient.GetGameResponsesByGameID(gameID)
	if errResponses != nil {
		log.Printf("handleManageResponses: Помилка отримання відгуків на гру %d: %v", gameID, errResponses)
		ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, "Помилка завантаження списку відгуків."))
		return
	}

	var msgText strings.Builder
	msgText.WriteString(fmt.Sprintf("*Відгуки на вашу гру:*\n_%s_\n\n", game.String()))
	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	if len(responses) == 0 {
		msgText.WriteString("Наразі немає відгуків.")
	} else {
		msgText.WriteString("Оберіть гравця для підтвердження гри:\n")
		for _, resp := range responses {
			if resp.Responder.UserID == 0 {
				log.Printf("handleManageResponses: Не вдалося завантажити дані гравця %d для відгуку %d", resp.ResponderID, resp.ID)
				msgText.WriteString(fmt.Sprintf("- Гравець ID %d (помилка завантаження даних)\n", resp.ResponderID))
				continue
			}
			responder := resp.Responder
			msgText.WriteString(fmt.Sprintf("👤 %s (@%s, R: %.0f)\n", responder.NameSurname, strings.TrimPrefix(responder.UserName, "@"), responder.Rating))
			chooseCallback := fmt.Sprintf("choose_opponent:%d:%d", gameID, responder.UserID)
			buttonText := fmt.Sprintf("✅ Обрати %s", responder.NameSurname)
			keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(buttonText, chooseCallback)))
		}
	}

	cancelCallback := fmt.Sprintf("cancel_proposal:%d", gameID)
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("❌ Скасувати пропозицію", cancelCallback)))

	msg := tgbotapi.NewMessage(proposerChatID, msgText.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}
	_, errSend := ev_proc.sendMessage(msg)
	if errSend != nil {
		log.Printf("handleManageResponses: Помилка надсилання списку відгуків пропозеру %d: %v", proposerID, errSend)
	}
}

func (ev_proc *EventProcessor) handleChooseOpponent(proposerChatID, proposerID int64, gameID uint, chosenResponderID int64, dbClient *db.DBClient) {
	log.Printf("handleChooseOpponent: Пропозер %d обрав гравця %d для гри %d", proposerID, chosenResponderID, gameID)

	game, errGame := dbClient.GetGame(gameID)
	if errGame != nil {
		log.Printf("handleChooseOpponent: Помилка отримання гри %d: %v", gameID, errGame)
		msgText := "Помилка: Не вдалося знайти гру для підтвердження."
		if errors.Is(errGame, gorm.ErrRecordNotFound) {
			msgText = "На жаль, ця гра вже неактуальна або була видалена."
		}
		ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, msgText))
		return
	}
	if game.Player.UserID != proposerID {
		log.Printf("handleChooseOpponent: Гравець %d намагається обрати суперника для чужої гри %d", proposerID, gameID)
		ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, "Ви не можете керувати цією грою."))
		return
	}

	chosenResponder, errChosen := dbClient.GetPlayer(chosenResponderID)
	if errChosen != nil {
		log.Printf("handleChooseOpponent: Помилка отримання даних обраного гравця %d: %v", chosenResponderID, errChosen)
		ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, "Помилка отримання даних обраного гравця."))
		return
	}
	proposer := game.Player

	allResponses, errResponses := dbClient.GetGameResponsesByGameID(gameID)
	if errResponses != nil {
		log.Printf("handleChooseOpponent: Помилка отримання всіх відгуків для гри %d: %v", gameID, errResponses)
	}

	msgToChosen := fmt.Sprintf("🎉 Вашу участь у грі підтверджено!\n\nГра: %s\nПропозер: %s (@%s)\n\nЗв'яжіться для узгодження деталей!",
		game.String(), proposer.NameSurname, strings.TrimPrefix(proposer.UserName, "@"))
	_, errSendChosen := ev_proc.sendMessage(tgbotapi.NewMessage(chosenResponderID, msgToChosen))
	if errSendChosen != nil {
		log.Printf("handleChooseOpponent: Помилка надсилання підтвердження обраному гравцю %d: %v", chosenResponderID, errSendChosen)
		ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, fmt.Sprintf("Не вдалося сповістити обраного гравця @%s. Спробуйте зв'язатися з ним самостійно.", strings.TrimPrefix(chosenResponder.UserName, "@"))))
		return
	}

	for _, resp := range allResponses {
		// Потрібно отримати повні дані гравця, що відгукнувся
		responder, errResponderInfo := dbClient.GetPlayer(resp.Responder.UserID) // Використовуємо UserID (int64) з вже завантаженого Responder
		if errResponderInfo != nil {
			log.Printf("handleChooseOpponent: Could not get full info for responder %d: %v", resp.Responder.UserID, errResponderInfo)
			continue // Пропустити цього відповідача, якщо не вдалося отримати дані
		}
		if responder.UserID != chosenResponderID {
			otherResponderID := resp.ResponderID
			if resp.Responder.UserID == 0 {
				log.Printf("handleChooseOpponent: Не вдалося завантажити дані гравця %d для сповіщення про відмову", resp.ResponderID)
				continue
			}
			otherResponderID_int64 := responder.UserID
			msgToOther := fmt.Sprintf("😕 На жаль, вашу заявку на гру '%s' з гравцем %s (@%s) було відхилено, оскільки обрано іншого суперника.",
				game.String(), proposer.NameSurname, strings.TrimPrefix(proposer.UserName, "@"))
			_, errSendOther := ev_proc.sendMessage(tgbotapi.NewMessage(otherResponderID_int64, msgToOther))
			if errSendOther != nil {
				log.Printf("handleChooseOpponent: Помилка надсилання відмови гравцю %d: %v", otherResponderID, errSendOther)
			}
		}
	}

	ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, fmt.Sprintf("✅ Ви обрали гравця %s (@%s) для гри:\n%s\n\nНе забудьте зв'язатися!",
		chosenResponder.NameSurname, strings.TrimPrefix(chosenResponder.UserName, "@"), game.String())))

	errDeleteGame := dbClient.DeleteGame(gameID)
	if errDeleteGame != nil && !errors.Is(errDeleteGame, gorm.ErrRecordNotFound) {
		log.Printf("handleChooseOpponent: Помилка видалення ProposedGame %d: %v", gameID, errDeleteGame)
	} else {
		log.Printf("handleChooseOpponent: ProposedGame %d видалено.", gameID)
	}

	deletedCount, errDeleteResponses := dbClient.DeleteGameResponsesByGameID(gameID)
	if errDeleteResponses != nil {
		log.Printf("handleChooseOpponent: Помилка видалення GameResponses для гри %d: %v", gameID, errDeleteResponses)
	} else {
		log.Printf("handleChooseOpponent: Видалено %d GameResponse записів для гри %d.", deletedCount, gameID)
	}

	ev_proc.mainMenu(proposerChatID)
}

func (ev_proc *EventProcessor) handleCancelProposal(proposerChatID, proposerID int64, gameID uint, dbClient *db.DBClient) {
	log.Printf("handleCancelProposal: Пропозер %d скасовує гру %d", proposerID, gameID)

	game, errGame := dbClient.GetGame(gameID)
	if errGame != nil {
		log.Printf("handleCancelProposal: Помилка отримання гри %d: %v", gameID, errGame)
		msgText := "Помилка: Не вдалося знайти гру для скасування."
		if errors.Is(errGame, gorm.ErrRecordNotFound) {
			msgText = "На жаль, ця гра вже неактуальна або була видалена."
		}
		ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, msgText))
		return
	}
	if game.Player.UserID != proposerID {
		log.Printf("handleCancelProposal: Гравець %d намагається скасувати чужу гру %d", proposerID, gameID)
		ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, "Ви не можете скасувати цю гру."))
		return
	}
	proposer := game.Player

	allResponses, errResponses := dbClient.GetGameResponsesByGameID(gameID)
	if errResponses != nil {
		log.Printf("handleCancelProposal: Помилка отримання відгуків для гри %d: %v", gameID, errResponses)
	}

	for _, resp := range allResponses {
		responderChatID := resp.Responder.UserID // resp.Responder має бути завантажений (GetGameResponsesByGameID робить Preload)
		if responderChatID == 0 {
			log.Printf("handleCancelProposal: Could not get UserID for responder ID %d", resp.ResponderID)
			continue
		}
		// === ВИДАЛЕНО НЕПОТРІБНЕ responder := resp.Responder ===
		msgToResponder := fmt.Sprintf("🚫 Гру '%s', запропоновану гравцем %s (@%s), на яку ви відгукувалися, було скасовано.",
			game.String(), proposer.NameSurname, strings.TrimPrefix(proposer.UserName, "@"))
		_, errSendOther := ev_proc.sendMessage(tgbotapi.NewMessage(responderChatID, msgToResponder))
		if errSendOther != nil {
			log.Printf("handleCancelProposal: Помилка надсилання сповіщення про скасування гравцю %d: %v", responderChatID, errSendOther) // Використовуємо responderChatID
		}
	}

	errDeleteGame := dbClient.DeleteGame(gameID)
	if errDeleteGame != nil && !errors.Is(errDeleteGame, gorm.ErrRecordNotFound) {
		log.Printf("handleCancelProposal: Помилка видалення ProposedGame %d: %v", gameID, errDeleteGame)
		ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, "Сталася помилка при видаленні гри."))
		return
	} else {
		log.Printf("handleCancelProposal: ProposedGame %d видалено.", gameID)
	}

	deletedCount, errDeleteResponses := dbClient.DeleteGameResponsesByGameID(gameID)
	if errDeleteResponses != nil {
		log.Printf("handleCancelProposal: Помилка видалення GameResponses для гри %d: %v", gameID, errDeleteResponses)
	} else {
		log.Printf("handleCancelProposal: Видалено %d GameResponse записів для гри %d.", deletedCount, gameID)
	}

	ev_proc.sendMessage(tgbotapi.NewMessage(proposerChatID, fmt.Sprintf("✅ Вашу пропозицію гри '%s' скасовано.", game.String())))
	ev_proc.mainMenu(proposerChatID)
}
