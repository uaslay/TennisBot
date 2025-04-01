// Package eventprocessor : This file contains the functions that handle the profile buttons.
package eventprocessor

import (
	"io"
	"os"
	"fmt"
	"log"
	"time"
	"strconv"
	"strings"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	db "TennisBot/database"
	"TennisBot/ui"
)

// TODO: error management

// ProfileButtonHandler is a function that handles the profile button.
func (ev_proc EventProcessor) ProfileButtonHandler(bot *tgbotapi.BotAPI, chatID int64, playerID int64, dbClient *db.DBClient) {
	// TODO: uncomment before final test
	player := dbClient.GetPlayer(playerID)

	msg := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(player.AvatarPhotoPath))
	msg.Caption = player.String()

	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}
	// FIXME: use send msg function
	editButtons := tgbotapi.NewMessage(chatID, ui.EditMsgMenu)

	editButtons.ReplyMarkup = ui.ProfileEditButtonOption
	ev_proc.bot.Send(editButtons)
}

// TODO: error management

// ProfilePhotoEditButtonHandler is a function that handles the profile photo edit button.
func (ev_proc EventProcessor) ProfilePhotoEditButtonHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), playerID int64, dbClient *db.DBClient) {
	state := ui.EditPhotoRequest
	player := dbClient.GetPlayer(playerID)
	chatID := update.CallbackQuery.From.ID

	activeRoutines[player.UserID] = make(chan string, 1)
	activeRoutines[player.UserID] <- update.CallbackQuery.Data

	for inputData := range activeRoutines[player.UserID] {
		if inputData == ui.QuitChannelCommand {
			break
		}

		switch state {
		case ui.EditPhotoRequest:
			// FIXME: use send msg function
			ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.EditMsgPhotoRequest))
			state = ui.EditPhotoResponse
		case ui.EditPhotoResponse:
			url, _ := bot.GetFileDirectURL(inputData)

			resp, err := http.Get(url)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			_ = os.Mkdir(PhotoFolderPath, os.ModePerm)

			out, err := os.Create(player.AvatarPhotoPath)
			if err != nil {
				panic(err)
			}
			defer out.Close()

			_, err = io.Copy(out, resp.Body)
			if err != nil {
				panic(err)
			}

			stopRoutine(player.UserID, activeRoutines)
			ev_proc.ProfileButtonHandler(bot, player.UserID, player.UserID, dbClient)
			// break
		}
	}
}

// TODO: error management

// ProfileRacketEditButtonHandler is a function that handles the profile name edit button.
func (ev_proc EventProcessor) ProfileRacketEditButtonHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), playerID int64, dbClient *db.DBClient) {
	state := ui.EditRacketRequest
	player := dbClient.GetPlayer(playerID)
	chatID := update.CallbackQuery.From.ID

	activeRoutines[player.UserID] = make(chan string, 1)
	activeRoutines[player.UserID] <- update.CallbackQuery.Data

	timer := time.NewTimer(ui.TimerPeriod)

out:
	for {
		select {
		case <-timer.C:
			log.Println("timer worked")
			delete(activeRoutines, player.UserID)
			break out
		case inputData := <-activeRoutines[player.UserID]:
			if inputData == ui.QuitChannelCommand {
				break out
			}

			switch state {
			case ui.EditRacketRequest:
				// FIXME: use send msg function
				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.EditMsgRacketRequest))
				state = ui.EditRacketResponse
			case ui.EditRacketResponse:
				dbClient.UpdatePlayer(inputData, playerID)
				stopRoutine(player.UserID, activeRoutines)
				ev_proc.ProfileButtonHandler(bot, player.UserID, player.UserID, dbClient)
				delete(activeRoutines, playerID)
				break out
			}
		}
	}
}

// DeleteGames is a function that handles the delete games button.
func (ev_proc EventProcessor) DeleteGames(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), playerID int64, dbClient *db.DBClient) {
	state := ui.ListOfGames
	player := dbClient.GetPlayer(playerID)
	chatID := update.CallbackQuery.From.ID

	activeRoutines[player.UserID] = make(chan string, 1)
	activeRoutines[player.UserID] <- update.CallbackQuery.Data

	timer := time.NewTimer(ui.TimerPeriod)

	// var messageId int
	// var replyMarkupMainMenu tgbotapi.InlineKeyboardMarkup

out:
	for {
		select {
		case <-timer.C:
			log.Println("timer worked")
			delete(activeRoutines, player.UserID)
			break out
		case inputData := <-activeRoutines[player.UserID]:
			if inputData == ui.QuitChannelCommand {
				break out
			}

			switch state {
			case ui.ListOfGames:
				games := dbClient.GetGamesByUserID(playerID)

				var replyMarkupMainMenu tgbotapi.InlineKeyboardMarkup
				for _, game := range games {
					replyMarkupMainMenu.InlineKeyboard = append(
						replyMarkupMainMenu.InlineKeyboard,
						tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(game.String(), fmt.Sprint(game.ID))))
				}
				// replyMarkupMainMenu.InlineKeyboard = append(
				// 	replyMarkupMainMenu.InlineKeyboard,
				// 	tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(ui.ProposeGame, ui.ProposeGame)))

				// TODO: const
				if len(replyMarkupMainMenu.InlineKeyboard) != 0 {
					// FIXME: use send msg function
					msg := tgbotapi.NewMessage(chatID, "Будь-ласка виберіть яку гру слід видалити")
					msg.ReplyMarkup = replyMarkupMainMenu
					_, err := ev_proc.bot.Send(msg)
					if err != nil {
						log.Panic(err)
					}
					// messageId = response.MessageID
					// log.Println(messageId)
					// activeRoutines[player.UserID] <- ""
					state = ui.DeleteGame
				} else {
					// FIXME: use send msg function
					msg := tgbotapi.NewMessage(chatID, "У вас не існує актуальних матчів")
					_, err := ev_proc.bot.Send(msg)
					if err != nil {
						log.Panic(err)
					}
				}
			case ui.DeleteGame:
				gameID, err := strconv.ParseUint(inputData, 0, 64)
				if err != nil {
					// log.Panic(err)
					continue
				}
				dbClient.DeleteGame(uint(gameID))
				state = ui.ListOfGames

				activeRoutines[player.UserID] <- ""
			}
		}
	}
}

// TODO: if someone else wanted to play

// // EnterGameScore is a function that handles the game score button.
// func (ev_proc EventProcessor) EnterGameScore(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), playerID int64, dbClient *db.DBClient) {
// 	// Score := []string{"1", "2", "3"}
// 	dbClient.DB.Create(&db.DualGame{ProposedPlayerID: 1, RespondedPlayerID: 1, ConfirmationProposed: true, ConfirmationResponded: true, BothConfirmed: false})

// 	// dbClient.DB.Create(&db.DualGame{ProposedPlayerID: 1, RespondedPlayerID: 1, ConfirmationProposed: true, ConfirmationResponded: true, Score: pg.StringArray(Score)})

// 	var res db.DualGame

// 	dbClient.DB.Where("proposed_player_id = ?", 1).First(&res)

// 	log.Println(res.Score[0])
// 	log.Println(res.Score[1])

// 	log.Println(res.Score[2])

// 	log.Println(res.Score[3])

// 	// Use proposedGames and dualGames (1 vs 1, 2 vs 2 == game type)
// 	// use only uconfirmed games
// 	// when proposed game confirmed, dualGame to be created
// 	// sets / games in game
// 	// fill game data set by set
// 	// output player - player: 1[6:4], 2[6:5]
// 	// if data filled, do not fill it
// 	// enter in cycle set1, games in format 6 4 == validate input
// 	// send confirmation request to the opponent
// 	// if score confirmed by both sides, proposed game to be deleted
// 	// list of games -> list of dual games for the correspondent user/player id
// 	// send request to confirm the game score
// }

// ScoreSubmitButtonHandler обробляє натискання кнопки "Зафіксувати рахунок".
// Очікується, що дані callback мають формат: "score:playerAID:playerBID:result"
// де result = "1" якщо перемога гравця A, або "0" якщо поразка.
func (ev_proc EventProcessor) ScoreSubmitButtonHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, dbClient *db.DBClient) {
	if update.CallbackQuery == nil {
		return
	}
	data := update.CallbackQuery.Data
	chatID := update.CallbackQuery.Message.Chat.ID

	// Розділяємо дані за роздільником ":"
	parts := strings.Split(data, ":")
	if len(parts) < 4 {
		log.Println("Невірний формат даних для фіксації рахунку")
		return
	}
	if parts[0] != "score" {
		log.Println("Невірний префікс даних, очікується 'score', отримано:", parts[0])
		return
	}
	playerAID := parts[1]
	playerBID := parts[2]
	result, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		log.Println("Помилка перетворення результату:", err)
		return
	}

	// Оновлюємо рейтинг гравців використовуючи функцію з ui/elo.go
	ui.UpdatePlayerRating(playerAID, playerBID, result)

	// Надсилаємо повідомлення з підтвердженням
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Рахунок зафіксовано. Рейтинг оновлено."))
	if _, err := bot.Send(msg); err != nil {
		log.Println("Помилка надсилання повідомлення:", err)
	}
}

// Обробляє кнопку "Загальний рейтинг"
func handleGeneralRating(bot *tgbotapi.BotAPI, chatID int64, userID string) {
    ratingMessage := ui.GetPlayerRating(userID)

    msg := tgbotapi.NewMessage(chatID, ratingMessage)
    bot.Send(msg)
}

var activeRoutines = make(map[int64]chan string)

// Обробляє кнопку "Зафіксувати рахунок"
func HandleFixScore(bot *tgbotapi.BotAPI, chatID int64, playerID int64, dbClient *db.DBClient, activeRoutines map[int64]chan string) {
	playerIDStr := fmt.Sprintf("%d", playerID)
	players := ui.LoadPlayers()
	player, exists := players[playerIDStr]

	if !exists || len(player.ActiveMatches) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "У вас немає активних матчів. Будь ласка, узгодьте гру з суперником перед фіксацією результату."))
		return
	}

	fmt.Printf("DEBUG: ActiveMatches for player %s: %+v\n", playerIDStr, player.ActiveMatches)

	if activeRoutines[playerID] != nil {
		close(activeRoutines[playerID])
	}
	activeRoutines[playerID] = make(chan string, 1)

	// Створюємо клавіатуру з активними матчами
	var buttons []tgbotapi.KeyboardButton
	for _, matchID := range player.ActiveMatches {
		buttons = append(buttons, tgbotapi.NewKeyboardButton(matchID))
	}

	keyboard := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(buttons...))
	msg := tgbotapi.NewMessage(chatID, "Оберіть матч для фіксації рахунку:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)

	// Чекаємо вибору матчу
	matchID := <-activeRoutines[playerID]

	// Визначаємо ID суперника з вибраного матчу
	opponentID := matchID
	_, opponentExists := players[opponentID]

	if !opponentExists {
		bot.Send(tgbotapi.NewMessage(chatID, "Помилка: не вдалося знайти суперника за обраним матчем."))
		return
	}

	// Вибір результату гри
	resultKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Перемога ✅"),
			tgbotapi.NewKeyboardButton("Поразка ❌"),
		),
	)
	resultMsg := tgbotapi.NewMessage(chatID, "Оберіть результат гри:")
	resultMsg.ReplyMarkup = resultKeyboard
	bot.Send(resultMsg)

	resultStr := <-activeRoutines[playerID]

	var result float64
	if resultStr == "Перемога ✅" {
		result = 1
	} else if resultStr == "Поразка ❌" {
		result = 0
	} else {
		bot.Send(tgbotapi.NewMessage(chatID, "Некоректний вибір. Будь ласка, оберіть 'Перемога ✅' або 'Поразка ❌'."))
		return
	}

	// Відправляємо підтвердження супернику
	opponentChatID, err := strconv.ParseInt(opponentID, 10, 64)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Помилка: некоректний ID суперника."))
		return
	}

	if activeRoutines[opponentChatID] != nil {
		close(activeRoutines[opponentChatID])
	}
	activeRoutines[opponentChatID] = make(chan string, 1)

	confirmMsg := tgbotapi.NewMessage(opponentChatID, fmt.Sprintf("Гравець @%s подав результат вашої гри як '%s'. Ви підтверджуєте?", player.Username, resultStr))
	confirmKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✅ Підтвердити"),
			tgbotapi.NewKeyboardButton("❌ Відхилити"),
		),
	)
	confirmMsg.ReplyMarkup = confirmKeyboard
	bot.Send(confirmMsg)

	// Очікуємо відповіді суперника
	confirmation := <-activeRoutines[opponentChatID]

	if confirmation == "✅ Підтвердити" {
		ui.UpdatePlayerRating(playerIDStr, opponentID, result)
		newRating := ui.GetPlayerRating(playerIDStr)
		bot.Send(tgbotapi.NewMessage(chatID, "Рахунок успішно зафіксовано!\n"+newRating))
		bot.Send(tgbotapi.NewMessage(opponentChatID, "Ви підтвердили результат гри. Рахунок оновлено."))
	} else {
		bot.Send(tgbotapi.NewMessage(chatID, "Опонент не підтвердив результат. Спробуйте ще раз."))
		bot.Send(tgbotapi.NewMessage(opponentChatID, "Ви відхилили результат гри."))
	}

	delete(activeRoutines, playerID)
	delete(activeRoutines, opponentChatID)
}

