package eventprocessor

import (
	ui "TennisBot/ui"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)
// OneTimeGameChoice is a struct that represents a single game choice
type OneTimeGameChoice struct {
	Area    string
	Date    string
	Time    string
	Partner string
	Payment string
	Court   string
}
// DefaultOneTimeGameChoice returns a default OneTimeGameChoice
func DefaultOneTimeGameChoice() OneTimeGameChoice {
	return OneTimeGameChoice{
		Area:    "",
		Date:    ui.DateToday,
		Time:    ui.TimeDoNotCare,
		Partner: ui.PartnerSingleManVsMan,
		Payment: ui.PaymentHalf,
		Court:   ui.CourtDoNotCare,
	}
}
// Serialize returns a string representation of a OneTimeGameChoice
func (s OneTimeGameChoice) Serialize() string {
	unixTimestamp, _ := strconv.ParseInt(s.Date, 10, 64)
	unixTime := time.Unix(unixTimestamp, 0)
	date := ConvertDayToUkr(int(unixTime.Weekday())) + " " + strconv.Itoa(unixTime.Day())

	result := "Вибрана дата:	" + date + "\n" + "\n" +
		"Вибраний час:	" + s.Time + "\n" + " \n" +
		"Тип гри:		" + s.Partner + "\n" + "\n" +
		"Оплата:		" + s.Payment + "\n" + "\n" +
		"Район:			" + s.Area + "\n" + "\n" +
		"Корт чи додаткова інформація: " + s.Court + "\n"
	return result
}

func processGameChoice(choice string,
	replyMarkup tgbotapi.InlineKeyboardMarkup,
	OneTimeGameChoice *OneTimeGameChoice) (tgbotapi.InlineKeyboardMarkup, error) {

	callback := strings.Split(choice, ":")

	if len(callback) != 2 {
		return tgbotapi.InlineKeyboardMarkup{}, fmt.Errorf("SingleGame, not valid input: %s", choice)
	}

	areaStatus := strings.Split(callback[0], "_")
	area, err := strconv.Atoi(areaStatus[1])

	if err != nil {
		log.Panic(err)
	}

	rowCol := strings.Split(callback[1], "_")
	row, _ := strconv.Atoi(rowCol[0])
	col, _ := strconv.Atoi(rowCol[1])

	if area == ui.RegionSection {
		status := areaStatus[0]

		if status == "x" {
			choiceLocal := strings.Replace(choice, "x", "a", 1)
			callback := ui.KeyboardCallback[ui.KeyboardData[choiceLocal]]
			callback = strings.Replace(callback, "x", "a", 1)
			replyMarkup.InlineKeyboard[row][col] = tgbotapi.NewInlineKeyboardButtonData(
				ui.KeyboardData[choiceLocal],
				callback,
			)
			OneTimeGameChoice.Area = strings.Replace(OneTimeGameChoice.Area, ui.KeyboardData[choiceLocal], " ", -1)
		} else {
			callback := ui.KeyboardCallback[ui.KeyboardData[choice]]
			callback = strings.Replace(callback, "a", "x", 1)
			replyMarkup.InlineKeyboard[row][col] = tgbotapi.NewInlineKeyboardButtonData(
				ui.Ball+ui.KeyboardData[choice],
				callback,
			)
			OneTimeGameChoice.Area = OneTimeGameChoice.Area + " " + ui.KeyboardData[choice]
		}
	} else {
		replyMarkup.InlineKeyboard[row][col] = tgbotapi.NewInlineKeyboardButtonData(
			ui.TextInversionMap[choice],
			ui.CallbackInversionMap[choice],
		)

		if area == ui.Date {
			OneTimeGameChoice.Date = ui.TextInversionMap[choice]
		} else if area == ui.Time {
			OneTimeGameChoice.Time = ui.TextInversionMap[choice]
		} else if area == ui.Partner {
			OneTimeGameChoice.Partner = ui.TextInversionMap[choice]
		} else if area == ui.Court {
			OneTimeGameChoice.Court = ui.TextInversionMap[choice]
		} else if area == ui.Payment {
			OneTimeGameChoice.Payment = ui.TextInversionMap[choice]
		}
	}
	return replyMarkup, nil
}
