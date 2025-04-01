package ui

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// MainKeyboard is a keyboard that represents the main menu
var (
	MainKeyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(FixScoreButton),
			tgbotapi.NewKeyboardButton("🎾 Разова гра"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(ProfileButton),
			tgbotapi.NewKeyboardButton(GeneralRatingButton),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🥇 Турніри"),
			tgbotapi.NewKeyboardButton("👍 Допомога"),
		),
	)
)
