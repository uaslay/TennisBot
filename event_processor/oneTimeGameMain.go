package eventprocessor

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

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
	player := dbClient.GetPlayer(playerID)
	chatID := update.Message.Chat.ID

	activeRoutines[player.UserID] = make(chan string, 1)
	activeRoutines[player.UserID] <- update.Message.Text

	replyMarkup := ui.NewKeyboard()

	timer := time.NewTimer(ui.TimerPeriod)

out:
	for {
		select {
		case <-timer.C:
			// TODO: const
			// FIXME: use send msg function
			msg := tgbotapi.NewMessage(chatID, "Будь-ласка перезайдіть в меню разової гри. Час використання меню сплив.")
			_, err := ev_proc.bot.Send(msg)
			if err != nil {
				log.Panic(err)
			}
			delete(activeRoutines, playerID)
			break out
		case inputData := <-activeRoutines[player.UserID]:
			if inputData == ui.QuitChannelCommand {
				delete(activeRoutines, playerID)
				break out
			}

			switch state {
			case ui.SingleGameMenu:
				// // log.Println("input ---- SingleGameMenu", inputData)
				if len(replyMarkupMainMenu.InlineKeyboard) > 0 {
					replyMarkupMainMenu.InlineKeyboard = nil
				}
				games := dbClient.GetGames()
				for _, game := range games {
					year, month, date := time.Now().Date()

					unixTimestamp, _ := strconv.ParseInt(game.Date, 10, 64)
					gameYear, gameMonth, gameDate := time.Unix(unixTimestamp, 0).Date()

					if gameYear >= year && gameMonth >= month && gameDate >= date {
						replyMarkupMainMenu.InlineKeyboard = append(
							replyMarkupMainMenu.InlineKeyboard,
							tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(game.String(), fmt.Sprint(game.ID))))
					}
				}
				replyMarkupMainMenu.InlineKeyboard = append(
					replyMarkupMainMenu.InlineKeyboard,
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(ui.ProposeGame, ui.ProposeGame)))
				// FIXME: use send msg function
				msg := tgbotapi.NewMessage(chatID, ui.InitialMessage)
				msg.ReplyMarkup = replyMarkupMainMenu
				response, err := ev_proc.bot.Send(msg)
				if err != nil {
					log.Panic(err)
				}
				messageID = response.MessageID
				activeRoutines[player.UserID] <- ""
				state = ui.ProcessSingleGameMenu
			case ui.ProcessSingleGameMenu:
				// log.Println("input ---- ProcessSingleGameMenu", inputData)

				if inputData == ui.ProposeGame {
					state = ui.ProposeGameMenu
					activeRoutines[player.UserID] <- ""
				} else {
					// 1. game was chosen
					if inputData != "" {

						// 2. show info about the player
						gameID, err := strconv.ParseUint(inputData, 0, 64)
						if err != nil {
							// log.Panic(err)
							continue
						}

						gamePlayer := dbClient.GetGameID(uint(gameID))
						if gamePlayer != 0 {
							player := dbClient.GetPlayer(gamePlayer)
							// FIXME: use send msg function
							msg := tgbotapi.NewPhoto(playerID, tgbotapi.FilePath(player.AvatarPhotoPath))
							msg.Caption = "Дані гравця:\n\n" + player.String()

							if _, err := bot.Send(msg); err != nil {
								panic(err)
							}

							// replyMarkup
							// TODO: const
							msgReply := tgbotapi.NewMessage(playerID, "Підтвержуєте ?")
							msgReply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
								tgbotapi.NewInlineKeyboardRow(
									// From whom:
									tgbotapi.NewInlineKeyboardButtonData(ui.GameConfirmationYes, inputData),
									tgbotapi.NewInlineKeyboardButtonData(ui.GameConfirmationNo, "nope"),
								),
							)
							// FIXME: use send msg function
							_, err = ev_proc.bot.Send(msgReply)
							if err != nil {
								log.Panic(err)
							}
							state = ui.GameWasChosen
						} else {
							// TODO: const
							// FIXME: use send msg function
							msg := tgbotapi.NewMessage(chatID, "Гра була щойно видалена гравцем. Будь-ласка зайдіть в меню знову.")
							_, err := ev_proc.bot.Send(msg)
							if err != nil {
								log.Panic(err)
							}
							state = ui.ProcessSingleGameMenu
						}
					}
				}
			case ui.GameWasChosen:
				// log.Println("input ---- GameWasChosen", inputData)
				if inputData != "" && inputData != "nope" {
					// 3. confirmation from the user about the game
					gameID, err := strconv.ParseUint(inputData, 0, 64)
					if err != nil {
						// log.Panic(err)
						// // log.Println("input ---- ", inputData)
						state = ui.ProcessSingleGameMenu
						continue
						// break
					}

					gamePlayer := dbClient.GetGameID(uint(gameID))
					log.Println(gamePlayer)

					if gamePlayer != 0 {
						// log.Println("gamePlayer->",gamePlayer)
						game := dbClient.GetGame(uint(gameID))
						// TODO: const
						// FIXME: use send msg function
						msg := tgbotapi.NewMessage(gamePlayer, "Відгук на Вашу гру:\n"+game.String())
						if _, err := bot.Send(msg); err != nil {
							log.Panic(err)
						}

						player := dbClient.GetPlayer(playerID)

						msgPlayerDetails := tgbotapi.NewPhoto(gamePlayer, tgbotapi.FilePath(player.AvatarPhotoPath))
						msgPlayerDetails.Caption = player.String()

						// FIXME: use send msg function
						if _, err := bot.Send(msgPlayerDetails); err != nil {
							log.Panic(err)
						}
						// FIXME: use send msg function
						msgReply := tgbotapi.NewMessage(gamePlayer, ui.GameProposal)
						msgReply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								// From whom:
								tgbotapi.NewInlineKeyboardButtonData(ui.GameConfirmationYes, ui.GameConfirmationYes+":"+strconv.FormatInt(player.UserID, 10)+":"+strconv.FormatInt(int64(game.ID), 10)),
								tgbotapi.NewInlineKeyboardButtonData(ui.GameConfirmationNo, ui.GameConfirmationNo+":"+strconv.FormatInt(player.UserID, 10)+":"+strconv.FormatInt(int64(game.ID), 10)),
							),
						)
						_, err = ev_proc.bot.Send(msgReply)
						if err != nil {
							log.Panic(err)
						}
					}
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
				if singleGameChoice.Area == "" {
					singleGameChoice.Area = "Не має значення"
				} else {
					fields := strings.Fields(singleGameChoice.Area)
					singleGameChoice.Area = strings.Join(fields, ", ")
				}

				if singleGameChoice.Date == ui.DateToday {
					singleGameChoice.Date = strconv.FormatInt(time.Now().Unix(), 10)
					// state = ui.ProcessDateChoice
				} else if singleGameChoice.Date == ui.DateTomorrow {
					singleGameChoice.Date = strconv.FormatInt(time.Now().Unix()+ui.Day, 10)
				}
				// FIXME: use send msg function
				msg := tgbotapi.NewMessage(chatID, "Все вірно ?")
				msg.Text = singleGameChoice.Serialize()
				msg.ReplyMarkup = ui.ChoiceConfirmation
				response, err := ev_proc.bot.Send(msg)

				if err != nil {
					log.Panic(err)
				} else {
					messageID = response.MessageID
				}

				state = ui.AllSelected
			case ui.AllSelected:
				if inputData == ui.Yes {
					// TODO: errors
					dbClient.CreateGame(db.ProposedGame{
						UserID:        playerID,
						RegionSection: singleGameChoice.Area,
						Partner:       singleGameChoice.Partner,
						Date:          singleGameChoice.Date,
						Time:          singleGameChoice.Time,
						Court:         singleGameChoice.Court,
						Payment:       singleGameChoice.Payment,
					})

					players := ui.LoadPlayers()
					playerA := players[fmt.Sprintf("%d", playerID)]
					playerB := players[fmt.Sprintf("%d", singleGameChoice.Partner)]

					// Створюємо унікальний ідентифікатор матчу
					matchID := fmt.Sprintf("%d_vs_%d", playerA.ID, playerB.ID)

					// Додаємо матч до активних матчів обох гравців
					playerA.ActiveMatches = append(playerA.ActiveMatches, matchID)
					playerB.ActiveMatches = append(playerB.ActiveMatches, matchID)

					// Оновлюємо дані гравців у мапі
					players[fmt.Sprintf("%d", playerA.ID)] = playerA
					players[fmt.Sprintf("%d", playerB.ID)] = playerB

					fmt.Printf("DEBUG: Players before save: %+v\n", players)
					
					// Зберігаємо зміни
					ui.SavePlayers(players)

					// TODO: const
					// FIXME: use send msg function
					msg := tgbotapi.NewMessage(chatID, "Гра зареєстрована")
					_, err := ev_proc.bot.Send(msg)
					if err != nil {
						log.Panic(err)
					}
				} else if inputData == ui.No {
					// TODO: const
					// FIXME: use send msg function
					msg := tgbotapi.NewMessage(chatID, "Реєстрація гри відмінена")
					_, err := ev_proc.bot.Send(msg)
					if err != nil {
						log.Panic(err)
					}
				}

				delete(activeRoutines, playerID)
				break out
			}
		}
	}
}
