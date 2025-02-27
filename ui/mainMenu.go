package ui

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// MainKeyboard is a keyboard that represents the main menu
var (
	MainKeyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✍️ Зафіксувати рахунок"),
			tgbotapi.NewKeyboardButton("🎾 Разова гра"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(ProfileButton),
			tgbotapi.NewKeyboardButton("📊 Загальний рейтинг"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🥇 Турніри"),
			tgbotapi.NewKeyboardButton("👍 Допомога"),
		),
	)
)
