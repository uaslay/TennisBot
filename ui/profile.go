package ui

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// List of input buttons
var (
	EditOptionPhoto  = "Фото"
	EditOptionRacket = "Ракетка"
	DeleteGames      = "Видалити гру"
	EnterGameScore   = "Внести рахунок"
)

// List of messages
var (
	EditMsgMenu            = "Можно відредагувати:"
	EditMsgRacketEditEntry = "Будь-ласка оберіть, що редагувати:"
	EditMsgRacketRequest   = "Будь-ласка, введіть виробника та назву ракетки."
	EditMsgPhotoRequest    = "Будь-ласка, відправте фото в чат.\nОпція компресіЇ фото повинна бути активною."
)

// List of input options
var (
	ProfileEditButtonOption = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(EditOptionPhoto, EditOptionPhoto),
			tgbotapi.NewInlineKeyboardButtonData(EditOptionRacket, EditOptionRacket),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(DeleteGames, DeleteGames),
			tgbotapi.NewInlineKeyboardButtonData(EnterGameScore, EnterGameScore),
		),
	)
)

// ProfileEditSteps is a type for profile edit steps
type ProfileEditSteps int16

// List of profile edit steps
const (
	EditRacketRequest ProfileEditSteps = iota
	EditRacketResponse
	EditPhotoRequest
	EditPhotoResponse
	ListOfGames
	DeleteGame
)
