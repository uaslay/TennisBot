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

	db "TennisBot/database"
	ui "TennisBot/ui"
)

// TODO: error management
func (ev_proc EventProcessor) registrationFlowHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, activeRoutines map[int64](chan string), dbClient *db.DBClient) {
	var err error

	state := ui.NameSurname
	player := db.Player{UserID: update.Message.From.ID}
	player.City = "КиЇв"
	chatID := update.Message.Chat.ID

	msg := tgbotapi.NewMessage(chatID, ui.ProfileRegistrationStart)
	_, err = ev_proc.bot.Send(msg)
	if err != nil {
		// log error
		panic(err)
	}

	activeRoutines[player.UserID] = make(chan string, 1)
	activeRoutines[player.UserID] <- update.Message.Text

	messageID := 0
	timer := time.NewTimer(ui.TimerPeriod)

out:
	for {
		select {
		case <-timer.C:
			// FIXME: use send msg function
			msg := tgbotapi.NewMessage(chatID, "Час очікування реєстраціЇ сплив. Тицніть на будь-який пункт меню.")
			_, err := ev_proc.bot.Send(msg)
			if err != nil {
				log.Panic(err)
			}
			delete(activeRoutines, player.UserID)
			break out
		case inputData := <-activeRoutines[player.UserID]:
			if inputData == "" {
				continue
			}

			if inputData == ui.QuitChannelCommand {
				break out
			}

			switch state {
			case ui.NameSurname:
				_, err := ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.ProfileMsgNameSurname))
				if err != nil {
					log.Panic(err)
				}
				state = ui.MobileNumber
			case ui.MobileNumber:
				// TODO: to const
				msg := tgbotapi.NewMessage(chatID, "Ваш номер телефона буде використаний виключно для комунікаціЇ з іншими користувачами данного чат бота при підтвердженні з Вашого боку участі в грі")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButtonContact("\xF0\x9F\x93\x9E Send phone"),
					),
				)
				response, err := bot.Send(msg)
				messageID = response.MessageID
				if err != nil {
					log.Panic(err)
				}

				player.NameSurname = inputData
				state = ui.Area
			case ui.Area:
				log.Println(inputData)
				input := strings.Split(inputData, ":")
				if len(input) != 2 {
					msg := tgbotapi.NewMessage(chatID, "Будь-ласка надайте номер телефона")
					msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButtonContact("\xF0\x9F\x93\x9E Send phone"),
						),
					)
					response, err := bot.Send(msg)
					messageID = response.MessageID
					if err != nil {
						log.Panic(err)
					}

					state = ui.Area
					continue
				}

				player.MobileNumber = "+" + input[0]
				player.UserName = "@" + input[1]
				// FIXME: use send msg function
				msgDelete := tgbotapi.NewDeleteMessage(chatID, messageID)
				_, err := ev_proc.bot.Request(msgDelete)
				if err != nil {
					log.Panic(err)
				}

				msg := tgbotapi.NewMessage(chatID, ui.ProfileMsgArea)
				msg.ReplyMarkup = ui.KyivRegions
				ev_proc.bot.Send(msg)
				state = ui.Age
			case ui.Age:
				player.Area = inputData
				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.ProfileMsgAge))
				state = ui.FromWhatAge
			case ui.FromWhatAge:
				player.YearOfBirth, err = strconv.Atoi(inputData)
				if err != nil {
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.ProfileMsgAge))
					continue
				}
				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.ProfileMsgFromWhatAge))
				state = ui.HowManyYears
			case ui.HowManyYears:
				player.YearStartedPlaying, err = strconv.Atoi(inputData)
				if err != nil {
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.ProfileMsgFromWhatAge))
					continue
				}
				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.ProfileMsgHowManyYearsInTennis))
				state = ui.ChampionshipsParticipation
			case ui.ChampionshipsParticipation:
				player.YearsOfPlayingWithoutInterrupts, err = strconv.Atoi(inputData)
				if err != nil {
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.ProfileMsgHowManyYearsInTennis))
					continue
				}
				msg := tgbotapi.NewMessage(chatID, ui.ProfileMsgHowManyParticipationInTournaments)
				msg.ReplyMarkup = ui.Championships
				ev_proc.bot.Send(msg)
				state = ui.FavouriteCourtMaterial
			case ui.FavouriteCourtMaterial:
				if inputData == "Так" {
					player.ChampionshipsParticipation = true
				} else {
					player.ChampionshipsParticipation = false
				}

				msg := tgbotapi.NewMessage(chatID, ui.ProfileFavouriteCourtMaterial)
				msg.ReplyMarkup = ui.FavourityCourtMaterial
				ev_proc.bot.Send(msg)
				state = ui.MainHand
			case ui.MainHand:
				fmt.Println(inputData)
				player.FavouriteCourt = inputData
				msg := tgbotapi.NewMessage(chatID, ui.ProfileMainHand)
				msg.ReplyMarkup = ui.MainGameHand
				ev_proc.bot.Send(msg)
				state = ui.RegistrationFinished
			case ui.RegistrationFinished:
				player.MainHand = inputData

				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.ProfileMsgSuccessfulRegistration))
				player.AvatarPhotoPath = fmt.Sprintf("%s%s", PhotoFolderPath, strconv.FormatInt(player.UserID, 10))
				stopRoutine(player.UserID, activeRoutines)

				dbClient.CreatePlayer(player)

				userProfilePhotos, err := bot.GetUserProfilePhotos(tgbotapi.UserProfilePhotosConfig{UserID: player.UserID})
				if err != nil {
					panic(err)
				}
				if userProfilePhotos.TotalCount != 0 {
					url, _ := bot.GetFileDirectURL(userProfilePhotos.Photos[0][len(userProfilePhotos.Photos[0])-1].FileID)

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
				}
				delete(activeRoutines, player.UserID)
				break out
			}
		}
	}
}
