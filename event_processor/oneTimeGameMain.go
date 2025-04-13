package eventprocessor

import (
	"fmt"
	"log"
	"time"
	"errors"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"

	db "TennisBot/database"
	ui "TennisBot/ui"
)

// TODO: error management

// OneTimeGameHandler is a handler for one-time game
func (ev_proc EventProcessor) OneTimeGameHandler(
	bot *tgbotapi.BotAPI,
	update tgbotapi.Update,
	activeRoutines map[int64](chan string),
	playerID int64,
	dbClient *db.DBClient) {

	var messageID int
	var replyMarkupMainMenu tgbotapi.InlineKeyboardMarkup
	dateChoiceFlag := false

	singleGameChoice := DefaultOneTimeGameChoice()

	state := ui.SingleGameMenu
	// ВИПРАВЛЕННЯ: Приймаємо 2 значення, додаємо обробку помилки
	player, errPlayer := dbClient.GetPlayer(playerID)
	if errPlayer != nil {
		log.Printf("OneTimeGameHandler: Помилка отримання гравця %d: %v", playerID, errPlayer)
		// Повідомляємо користувача, якщо не вдалося отримати дані
		if update.Message != nil {
			ev_proc.bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Помилка завантаження даних профілю."))
		}
		return // Не можемо продовжити без даних гравця
	}
	var chatID int64
	if update.Message != nil {
			chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
			chatID = update.CallbackQuery.Message.Chat.ID
	} else {
			log.Println("OneTimeGameHandler: Не вдалося визначити chatID")
			return
	}


	if _, exists := activeRoutines[player.UserID]; exists {
		log.Printf("OneTimeGameHandler: Рутина вже активна для користувача %d.", player.UserID)
		ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Будь ласка, завершіть попередню дію."))
		return
	}

	activeRoutines[player.UserID] = make(chan string, 1)

	// Надсилаємо тригер
	if update.Message != nil {
		activeRoutines[player.UserID] <- update.Message.Text
	} else if update.CallbackQuery != nil {
		// Якщо це callback, можливо, не потрібно надсилати data одразу?
		// Залежить від того, чи очікує початковий стан цей callback.
		// Поки що надсилаємо, як було.
		activeRoutines[player.UserID] <- update.CallbackQuery.Data
	}


	replyMarkup := ui.NewKeyboard()

	timer := time.NewTimer(ui.TimerPeriod)
	defer func() {
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
		case <-timer.C:
			log.Printf("OneTimeGameHandler: Таймер спрацював для користувача %d.", player.UserID)
			msg := tgbotapi.NewMessage(chatID, "Час очікування сплив. Будь ласка, спробуйте ще раз.")
			_, _ = ev_proc.bot.Send(msg) // Ігноруємо помилку тут
			break out
		case inputData, ok := <-activeRoutines[player.UserID]:
			if !ok {
				log.Printf("OneTimeGameHandler: Канал для користувача %d закрито.", player.UserID)
				break out
			}
			if !timer.Stop() {
				select { case <-timer.C: default: }
			}
			timer.Reset(ui.TimerPeriod)

			if inputData == ui.QuitChannelCommand {
				log.Printf("OneTimeGameHandler: Отримано команду виходу для користувача %d.", player.UserID)
				break out
			}

			switch state {
			case ui.SingleGameMenu:
				// // log.Println("input ---- SingleGameMenu", inputData)
				if len(replyMarkupMainMenu.InlineKeyboard) > 0 {
					replyMarkupMainMenu.InlineKeyboard = nil
				}
				// ВИПРАВЛЕННЯ: Приймаємо 2 значення, додаємо обробку помилки
				games, errGames := dbClient.GetGames()
				if errGames != nil {
					log.Printf("Помилка отримання списку ігор: %v", errGames)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося завантажити список ігор."))
					break out
				}

				// ... (логіка фільтрації та створення клавіатури) ...
				currentTime := time.Now()
				todayStart := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())

				for _, game := range games {
					unixTimestamp, errParse := strconv.ParseInt(game.Date, 10, 64)
					if errParse != nil {
						log.Printf("Помилка парсингу дати гри %d: %v", game.ID, errParse)
						continue
					}
					gameTime := time.Unix(unixTimestamp, 0)

					if gameTime.After(todayStart) || gameTime.Equal(todayStart) {
						if game.UserID != playerID {
								replyMarkupMainMenu.InlineKeyboard = append(
									replyMarkupMainMenu.InlineKeyboard,
									tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(game.String(), fmt.Sprint(game.ID))),
								)
						}
					}
				}
				// ... (кінець логіки клавіатури) ...

				replyMarkupMainMenu.InlineKeyboard = append(
					replyMarkupMainMenu.InlineKeyboard,
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(ui.ProposeGame, ui.ProposeGame)),
				)
				// FIXME: use send msg function
				msg := tgbotapi.NewMessage(chatID, ui.InitialMessage)
				msg.ReplyMarkup = replyMarkupMainMenu
				response, err := ev_proc.bot.Send(msg)
				if err != nil {
					log.Panic(err)
				}
				messageID = response.MessageID
				state = ui.ProcessSingleGameMenu

			case ui.ProcessSingleGameMenu:
				if inputData == ui.ProposeGame {
					state = ui.ProposeGameMenu
					activeRoutines[player.UserID] <- ""
				} else if inputData != "" {
					gameID, err := strconv.ParseUint(inputData, 0, 64)
					if err != nil {
						log.Printf("Невірний gameID '%s': %v", inputData, err)
						state = ui.SingleGameMenu
						activeRoutines[player.UserID] <- ""
						continue
					}

					// ВИПРАВЛЕННЯ: Приймаємо 2 значення для GetGame
					game, errGame := dbClient.GetGame(uint(gameID))
					if errGame != nil {
						log.Printf("Помилка отримання гри %d: %v", gameID, errGame)
						// Перевірка на RecordNotFound
						if errors.Is(errGame, gorm.ErrRecordNotFound) {
							ev_proc.bot.Send(tgbotapi.NewMessage(chatID,"Гру не знайдено. Можливо, її вже видалено."))
						} else {
							ev_proc.bot.Send(tgbotapi.NewMessage(chatID,"Помилка завантаження гри."))
						}
						state = ui.SingleGameMenu
						activeRoutines[player.UserID] <- ""
						continue
					}

					gamePlayerID := game.UserID // ID гравця, що запропонував гру
					// ВИПРАВЛЕННЯ: Приймаємо 2 значення для GetPlayer
					gamePlayer, errGamePlayer := dbClient.GetPlayer(gamePlayerID)
					if errGamePlayer != nil {
						log.Printf("Помилка отримання даних гравця %d для гри %d: %v", gamePlayerID, gameID, errGamePlayer)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Не вдалося отримати дані гравця для цієї гри."))
						state = ui.SingleGameMenu
						activeRoutines[player.UserID] <- ""
						continue
					}

					// ... (надсилання фото/інфо гравця) ...
					var photoMsg tgbotapi.Chattable
					if gamePlayer.AvatarPhotoPath != "" {
						photo := tgbotapi.NewPhoto(playerID, tgbotapi.FilePath(gamePlayer.AvatarPhotoPath)) // Перевірити playerID чи chatID? Має бути chatID одержувача
						photo.Caption = "Дані гравця:\n\n" + gamePlayer.String()
						photoMsg = photo
					} else {
						msg := tgbotapi.NewMessage(playerID, "Дані гравця:\n\n"+gamePlayer.String()) // Перевірити playerID чи chatID? Має бути chatID одержувача
						photoMsg = msg
					}
					// Виправляємо ID чату для надсилання
					if pMsg, ok := photoMsg.(tgbotapi.PhotoConfig); ok { pMsg.ChatID = chatID; photoMsg = pMsg }
					if mMsg, ok := photoMsg.(tgbotapi.MessageConfig); ok { mMsg.ChatID = chatID; photoMsg = mMsg }

					if _, err := bot.Send(photoMsg); err != nil {
						log.Printf("Помилка надсилання даних гравця: %v", err)
					}
					// ... (кінець надсилання фото/інфо) ...


					msgReply := tgbotapi.NewMessage(chatID, "Бажаєте відгукнутися на цю гру?") // Надсилаємо в поточний чат
					confirmDataYes := fmt.Sprintf("confirm_game:yes:%d", gameID)
					confirmDataNo := fmt.Sprintf("confirm_game:no:%d", gameID)
					msgReply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData(ui.GameConfirmationYes, confirmDataYes),
							tgbotapi.NewInlineKeyboardButtonData(ui.GameConfirmationNo, confirmDataNo),
						),
					)
					_, err = ev_proc.bot.Send(msgReply)
					if err != nil {
						log.Printf("Помилка надсилання запиту підтвердження гри: %v", err)
					}
					// Стан не змінюємо, чекаємо callback
				}
			
			case ui.ProposeGameMenu:
				// log.Println("input ---- ProposeGameMenu", inputData)

				var msg tgbotapi.EditMessageReplyMarkupConfig
				var msgResponse tgbotapi.MessageConfig

				if messageID != 0 {
					msg = tgbotapi.NewEditMessageReplyMarkup(update.Message.Chat.ID, messageID, replyMarkup)
				} else {
					msgResponse = tgbotapi.NewMessage(chatID, ui.InitialMessage)
					singleGameChoice.Time = ui.TimeDoNotCare
					replyMarkup = ui.NewKeyboard()
					msgResponse.ReplyMarkup = replyMarkup
				}

				if messageID != 0 {
					// FIXME: use send msg function
					response, err := ev_proc.bot.Send(msg)
					if err != nil {
						log.Panic(err)
					} else {
						messageID = response.MessageID
					}
				} else {
					// FIXME: use send msg function
					response, err := ev_proc.bot.Send(msgResponse)
					if err != nil {
						log.Panic(err)
					} else {
						messageID = response.MessageID
					}
				}

				state = ui.EditProposeGameMenu
				activeRoutines[player.UserID] <- ""
			case ui.EditProposeGameMenu:
				// log.Println("input ---- EditProposeGameMenu ", inputData)

				if inputData == ui.Ok {
					state = ui.DateChoice
					activeRoutines[player.UserID] <- ""
				} else if inputData == ui.Back {
					state = ui.SingleGameMenu
					// FIXME: use send msg function
					msgDelete := tgbotapi.NewDeleteMessage(chatID, messageID)
					_, err := ev_proc.bot.Request(msgDelete)
					if err != nil {
						log.Panic(err)
					}
					// log.Println("input ---- 3 ", inputData)
					activeRoutines[player.UserID] <- ""
				} else {
					// log.Println("input ---- 4 ", inputData)

					if inputData != "" {
						// log.Println(singleGameChoice.Time, singleGameChoice.Date)
						replyMarkup, err := processGameChoice(inputData, replyMarkup, &singleGameChoice)
						if err != nil {
							log.Println(err)
							continue
						}
						// FIXME: use send msg function
						msg := tgbotapi.NewEditMessageReplyMarkup(update.Message.Chat.ID, messageID, replyMarkup)
						response, err := ev_proc.bot.Send(msg)

						if err != nil {
							log.Panic(err)
						} else {
							messageID = response.MessageID
						}
						state = ui.EditProposeGameMenu
					}
				}
			case ui.DateChoice:
				// log.Println("here")
				if singleGameChoice.Date == ui.DateWillSpecify {
					// log.Println("here -1 ")

					if messageID != 0 {
						// FIXME: use send msg function
						msg := tgbotapi.NewEditMessageReplyMarkup(update.Message.Chat.ID, messageID, GenerateCalendarForSingleGameChoice())
						response, err := ev_proc.bot.Send(msg)
						if err != nil {
							log.Panic(err)
						} else {
							messageID = response.MessageID
						}
					} else {
						// FIXME: use send msg function
						msgResponse := tgbotapi.NewMessage(chatID, ui.InitialMessage)
						msgResponse.ReplyMarkup = GenerateCalendarForSingleGameChoice()
						response, err := ev_proc.bot.Send(msgResponse)
						if err != nil {
							log.Panic(err)
						} else {
							messageID = response.MessageID
						}
					}

					state = ui.ProcessDateChoice
					dateChoiceFlag = true
				} else if singleGameChoice.Time == ui.TimeWillSpecify {
					// log.Println("here -2 ")

					state = ui.TimeChoice
					activeRoutines[player.UserID] <- ""
				} else {
					// log.Println("here -3 ")

					state = ui.ProcessTimeChoice
					activeRoutines[player.UserID] <- ""
				}
			case ui.ProcessDateChoice:
				if inputData == ui.Back {
					state = ui.ProposeGameMenu
					// FIXME: use send msg function
					msgDelete := tgbotapi.NewDeleteMessage(chatID, messageID)
					_, err := ev_proc.bot.Request(msgDelete)
					if err != nil {
						log.Panic(err)
					}
					messageID = 0
					singleGameChoice.Date = ui.DateToday
					dateChoiceFlag = false
				} else {
					singleGameChoice.Date = inputData
					state = ui.TimeChoice
				}

				activeRoutines[player.UserID] <- ""
			case ui.TimeChoice:
				state = ui.ProcessTimeChoice

				if singleGameChoice.Time == ui.TimeWillSpecify {
					if messageID != 0 {
						// FIXME: use send msg function
						msg := tgbotapi.NewEditMessageReplyMarkup(update.Message.Chat.ID, messageID, ui.NewTimeKeyboard(false))
						response, err := ev_proc.bot.Send(msg)
						if err != nil {
							log.Panic(err)
						} else {
							messageID = response.MessageID
						}
					} else {
						// FIXME: use send msg function
						msgResponse := tgbotapi.NewMessage(chatID, ui.InitialMessage)
						msgResponse.ReplyMarkup = ui.NewTimeKeyboard(false)
						response, err := ev_proc.bot.Send(msgResponse)
						if err != nil {
							log.Panic(err)
						} else {
							messageID = response.MessageID
						}
					}
				} else {
					activeRoutines[player.UserID] <- ""
				}
			case ui.ProcessTimeChoice:
				if inputData != ui.Back {
					if singleGameChoice.Time == ui.TimeWillSpecify {
						singleGameChoice.Time = inputData
						// FIXME: use send msg function
						msgDelete := tgbotapi.NewDeleteMessage(chatID, messageID)
						_, err := ev_proc.bot.Request(msgDelete)
						if err != nil {
							log.Panic(err)
						}
					}

					if singleGameChoice.Court == ui.CourtWillSpecify {
						// TODO: const
						// FIXME: use send msg function
						msg := tgbotapi.NewMessage(chatID, "Будь-ласка, вкажіть корт чи будь-яку іншу корисну інформацію")
						_, err := ev_proc.bot.Send(msg)
						if err != nil {
							log.Panic(err)
						}
						state = ui.SelectCourt
					} else {
						activeRoutines[player.UserID] <- ""
						state = ui.Selected
					}
				} else {
					// FIXME: use send msg function
					msgDelete := tgbotapi.NewDeleteMessage(chatID, messageID)
					_, err := ev_proc.bot.Request(msgDelete)
					if err != nil {
						log.Panic(err)
					}

					messageID = 0
					if dateChoiceFlag {
						singleGameChoice.Date = ui.DateWillSpecify
						state = ui.DateChoice
					} else {
						singleGameChoice.Date = ui.DateToday
						state = ui.ProposeGameMenu
					}
					dateChoiceFlag = false
					activeRoutines[player.UserID] <- ""
				}

			case ui.SelectCourt:
				singleGameChoice.Court = inputData
				state = ui.Selected
				activeRoutines[player.UserID] <- ""
			case ui.Selected:
				// ... (логіка перевірки Area) ...
				if singleGameChoice.Area == "" { singleGameChoice.Area = "Не вказано" } else { singleGameChoice.Area = strings.TrimSpace(singleGameChoice.Area)}

				// ... (логіка перевірки/встановлення Date) ...
				if singleGameChoice.Date == "" {
					if dateChoiceFlag || singleGameChoice.Date == ui.DateWillSpecify {
						log.Println("Помилка: Дата не була обрана після 'вкажу'.")
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Помилка: дата не була обрана."))
						state = ui.DateChoice
						activeRoutines[player.UserID] <- ""
						continue
					} else {
						singleGameChoice.Date = strconv.FormatInt(time.Now().Unix(), 10)
					}
				} else if singleGameChoice.Date == ui.DateToday {
					singleGameChoice.Date = strconv.FormatInt(time.Now().Unix(), 10)
				} else if singleGameChoice.Date == ui.DateTomorrow {
					singleGameChoice.Date = strconv.FormatInt(time.Now().AddDate(0, 0, 1).Unix(), 10)
				}

				// ... (видалення попереднього повідомлення) ...
				if messageID != 0 {
					msgDelete := tgbotapi.NewDeleteMessage(chatID, messageID)
					_, _ = ev_proc.bot.Request(msgDelete) // Ігноруємо помилку видалення
					messageID = 0
				}

				msg := tgbotapi.NewMessage(chatID, "Перевірте та підтвердіть вашу гру:")
				msg.Text = singleGameChoice.Serialize()
				msg.ReplyMarkup = ui.ChoiceConfirmation
				response, err := ev_proc.bot.Send(msg)
				if err != nil {
					log.Panic(err)
				}
				messageID = response.MessageID
				state = ui.AllSelected
			case ui.AllSelected:
				// ... (видалення повідомлення з кнопками) ...
				if messageID != 0 {
						msgDelete := tgbotapi.NewDeleteMessage(chatID, messageID)
						_, _ = ev_proc.bot.Request(msgDelete)
						messageID = 0
				}

				if inputData == ui.Yes {
					game := db.ProposedGame{ /* ... поля гри ... */ }
					// ВИПРАВЛЕННЯ: Обробляємо помилку CreateGame
					errCreate := dbClient.CreateGame(game)
					if errCreate != nil {
						log.Printf("Помилка створення гри в БД для гравця %d: %v", playerID, errCreate)
						msg := tgbotapi.NewMessage(chatID, "Помилка при збереженні гри. Спробуйте ще раз.")
						ev_proc.bot.Send(msg)
					} else {
						log.Printf("Гра успішно створена в БД для гравця %d", playerID)
						msg := tgbotapi.NewMessage(chatID, "Вашу гру зареєстровано!")
						ev_proc.bot.Send(msg)
					}
				} else if inputData == ui.No {
					msg := tgbotapi.NewMessage(chatID, "Створення гри скасовано.")
					ev_proc.bot.Send(msg)
				}
				log.Printf("OneTimeGameHandler: Завершення після AllSelected для %d.", playerID)
				break out

			} // end switch state
		} // end select
	} // end for
	log.Printf("OneTimeGameHandler: Вихід з функції для користувача %d.", playerID)
} // end func OneTimeGameHandler