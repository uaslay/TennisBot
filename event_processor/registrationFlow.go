package eventprocessor

import (
	"log"
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
		log.Printf("Помилка під час реєстрації (стан %v) для користувача %d: %v", state, player.UserID, err)
		ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Сталася помилка під час реєстрації. Спробуйте почати знову: /start"))
		delete(activeRoutines, player.UserID)
		return
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
				log.Printf("Помилка під час реєстрації (стан %v) для користувача %d: %v", state, player.UserID, err)
				ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Сталася помилка під час реєстрації. Спробуйте почати знову: /start"))
				delete(activeRoutines, player.UserID)
				break out
			}
			delete(activeRoutines, player.UserID)
			break out
		case inputData, ok := <-activeRoutines[player.UserID]: // Додаємо 'ok' для перевірки закриття
			if !ok { // Перевірка на закриття каналу
				log.Printf("RegistrationFlowHandler: Канал для %d закрито.", player.UserID)
				break out
			}
			// ... (timer reset) ...
			if inputData == ui.QuitChannelCommand { // Перевірка команди виходу
				log.Printf("RegistrationFlowHandler: Команда виходу для %d.", player.UserID)
				break out
			}

			switch state {
			case ui.NameSurname:
				_, err := ev_proc.bot.Send(tgbotapi.NewMessage(chatID, ui.ProfileMsgNameSurname))
				if err != nil {
					log.Printf("Помилка під час реєстрації (стан %v) для користувача %d: %v", state, player.UserID, err)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Сталася помилка під час реєстрації. Спробуйте почати знову: /start"))
					delete(activeRoutines, player.UserID)
					break out
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
					log.Printf("Помилка під час реєстрації (стан %v) для користувача %d: %v", state, player.UserID, err)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Сталася помилка під час реєстрації. Спробуйте почати знову: /start"))
					delete(activeRoutines, player.UserID)
					break out
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
						log.Printf("Помилка під час реєстрації (стан %v) для користувача %d: %v", state, player.UserID, err)
						ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Сталася помилка під час реєстрації. Спробуйте почати знову: /start"))
						delete(activeRoutines, player.UserID)
						break out
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
					log.Printf("Помилка під час реєстрації (стан %v) для користувача %d: %v", state, player.UserID, err)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Сталася помилка під час реєстрації. Спробуйте почати знову: /start"))
					delete(activeRoutines, player.UserID)
					break out
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
			case ui.MainHand: // Попередня відповідь (про покриття) оброблена, бот запитав про руку
				player.FavouriteCourt = inputData // Зберігаємо відповідь про покриття з попереднього кроку

				// Надсилаємо питання про ігрову руку
				msg := tgbotapi.NewMessage(chatID, ui.ProfileMainHand)
				msg.ReplyMarkup = ui.MainGameHand
				_, err = ev_proc.bot.Send(msg) // Виправляємо змінну помилки
				if err != nil {
					log.Printf("Помилка надсилання питання про руку: %v", err)
					// Обробка помилки, можливо break out
					break out
				}
				state = ui.RegistrationFinished // Переходимо в фінальний стан (АЛЕ відповідь прийде сюди)

			case ui.RegistrationFinished: // Сюди приходить відповідь "Права" або "Ліва"
				player.MainHand = inputData // Зберігаємо відповідь про руку

				// --- Одразу виконуємо фінальні дії ---
				finalMsg := tgbotapi.NewMessage(chatID, ui.ProfileMsgSuccessfulRegistration)
				finalMsg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true) // Прибираємо клавіатуру
				ev_proc.bot.Send(finalMsg)

				// Зберігаємо гравця в БД
				errDb := dbClient.CreatePlayer(player)
				if errDb != nil {
					log.Printf("Помилка реєстрації гравця %d: %v", player.UserID, errDb)
					ev_proc.bot.Send(tgbotapi.NewMessage(chatID, "Помилка під час реєстрації. Спробуйте пізніше."))
					break out
				} else {
					log.Printf("Гравець %d успішно зареєстрований", player.UserID)
				}

				// Автоматичне отримання та збереження FileID фото профілю ТГ
				log.Printf("Спроба отримати фото профілю ТГ для %d", player.UserID)
				userProfilePhotos, errPhotos := bot.GetUserProfilePhotos(tgbotapi.UserProfilePhotosConfig{UserID: player.UserID})
				if errPhotos != nil {
					log.Printf("Помилка отримання фото профілю ТГ для %d: %v", player.UserID, errPhotos)
				} else if userProfilePhotos.TotalCount > 0 && len(userProfilePhotos.Photos) > 0 && len(userProfilePhotos.Photos[0]) > 0 {
					fileID := userProfilePhotos.Photos[0][len(userProfilePhotos.Photos[0])-1].FileID
					log.Printf("Отримано FileID фото профілю ТГ для %d: %s. Оновлюємо БД.", player.UserID, fileID)
					errUpdate := dbClient.UpdatePlayer(player.UserID, map[string]interface{}{"AvatarFileID": fileID})
					if errUpdate != nil {
						log.Printf("Помилка оновлення AvatarFileID для %d під час реєстрації: %v", player.UserID, errUpdate)
					} else {
						log.Printf("AvatarFileID для гравця %d автоматично оновлено в БД.", player.UserID)
					}
				} else {
					log.Printf("У користувача %d немає фото профілю ТГ або їх не вдалося отримати.", player.UserID)
				}

				// Показуємо головне меню
				ev_proc.mainMenu(chatID)
				break out // Завершуємо рутину реєстрації
			}
		}
	}
	// Якщо рутина завершилася через таймер або помилку, переконуємося, що вона видалена з мапи
	if _, exists := activeRoutines[player.UserID]; exists {
		delete(activeRoutines, player.UserID)
		log.Printf("RegistrationFlowHandler: Рутина для %d видалена після завершення циклу.", player.UserID)
	}
}
