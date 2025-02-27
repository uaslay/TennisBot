// Package eventprocessor : This file contains the functions that handle the profile buttons.
package eventprocessor
package ui

import (
	"io"
	"os"
	"fmt"
	"log"
	"time"
	"strconv"
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

// EnterGameScore is a function that handles the game score button.
func (ev_proc EventProcessor) EnterGameScore(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), playerID int64, dbClient *db.DBClient) {
	// Score := []string{"1", "2", "3"}
	dbClient.DB.Create(&db.DualGame{ProposedPlayerID: 1, RespondedPlayerID: 1, ConfirmationProposed: true, ConfirmationResponded: true, BothConfirmed: false})

	// dbClient.DB.Create(&db.DualGame{ProposedPlayerID: 1, RespondedPlayerID: 1, ConfirmationProposed: true, ConfirmationResponded: true, Score: pg.StringArray(Score)})

	var res db.DualGame

	dbClient.DB.Where("proposed_player_id = ?", 1).First(&res)

	log.Println(res.Score[0])
	log.Println(res.Score[1])

	log.Println(res.Score[2])

	log.Println(res.Score[3])

	// Use proposedGames and dualGames (1 vs 1, 2 vs 2 == game type)
	// use only uconfirmed games
	// when proposed game confirmed, dualGame to be created
	// sets / games in game
	// fill game data set by set
	// output player - player: 1[6:4], 2[6:5]
	// if data filled, do not fill it
	// enter in cycle set1, games in format 6 4 == validate input
	// send confirmation request to the opponent
	// if score confirmed by both sides, proposed game to be deleted
	// list of games -> list of dual games for the correspondent user/player id
	// send request to confirm the game score
}

// Обробляє кнопку "Загальний рейтинг"
func handleGeneralRating(userID string) string {
    return getPlayerRating(userID)
}
