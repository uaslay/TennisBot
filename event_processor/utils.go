package eventprocessor

import (
	ui "TennisBot/ui"

	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func stopRoutine(playerID int64, activeRoutines map[int64](chan string)) {
	if activeRoutines[playerID] != nil {
		/* stop goroutine */
		activeRoutines[playerID] <- ui.QuitChannelCommand

		/* close correspondent channel */
		close(activeRoutines[playerID])

		/* erase allocated structures for channels */
		delete(activeRoutines, playerID)
	}
}

// ConvertDayToUkr is a function that converts the day of the week to Ukrainian.
func ConvertDayToUkr(day int) string {
	if day == 1 {
		return "Пн"
	} else if day == 2 {
		return "Вт"
	} else if day == 3 {
		return "Ср"
	} else if day == 4 {
		return "Чт"
	} else if day == 5 {
		return "Пт"
	} else if day == 6 {
		return "Сб"
	} else {
		return "Нд"
	}
}

// GenerateCalendarForSingleGameChoice is a function that generates a calendar for a single game choice.
func GenerateCalendarForSingleGameChoice() tgbotapi.InlineKeyboardMarkup {
	currentTime := time.Now()

	// TODO: cache on every new day, maybe goroutine
	var calendar [4][4]string
	var unixTimestamp [4][4]string
	k := 0

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			calendar[i][j] = ConvertDayToUkr(int(currentTime.AddDate(0, 0, k).Weekday())) + " " + strconv.Itoa(currentTime.AddDate(0, 0, k).Day())
			k++
		}
	}
	k = 0
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			unixTimestamp[i][j] = strconv.FormatInt(currentTime.AddDate(0, 0, k).Unix(), 10)
			// log.Println(unixTimestamp[i][j])
			k++
		}
	}

	CalendarKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(calendar[0][0], unixTimestamp[0][0]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[0][1], unixTimestamp[0][1]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[0][2], unixTimestamp[0][2]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[0][3], unixTimestamp[0][3]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(calendar[1][0], unixTimestamp[1][0]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[1][1], unixTimestamp[1][1]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[1][2], unixTimestamp[1][2]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[1][3], unixTimestamp[1][3]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(calendar[2][0], unixTimestamp[2][0]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[2][1], unixTimestamp[2][1]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[2][2], unixTimestamp[2][2]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[2][3], unixTimestamp[2][3]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(calendar[3][0], unixTimestamp[3][0]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[3][1], unixTimestamp[3][1]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[3][2], unixTimestamp[3][2]),
			tgbotapi.NewInlineKeyboardButtonData(calendar[3][3], unixTimestamp[3][3]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(ui.Back, ui.Back),
		),
	)

	return CalendarKeyboard
}
