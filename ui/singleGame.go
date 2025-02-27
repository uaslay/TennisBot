package ui

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SingleGameSteps is a type that represents the steps of the single game process
type SingleGameSteps int16

// Define constants for different sections of the single game process
const (
	RegionSection int = iota
	Partner
	Date
	Time
	Court
	Payment
)

// Define constants for different sections of the single game process
const (
	Ball        = "🎾"
	Ok          = "🆗"
	Back        = "Назад"
	ProposeGame = "Запропонувати гру"
)

// Define constants for different sections of the single game process
const (
	SingleGameMenu SingleGameSteps = iota
	ProcessSingleGameMenu
	ProposeGameMenu
	GameWasChosen
	EditProposeGameMenu
	DateChoice
	ProcessDateChoice
	TimeChoice
	ProcessTimeChoice
	AllSelected
	SelectCourt
	Selected
)

// Define constants for different sections of the single game process
const (
	PartnerSingleManVsMan   = "🙍‍♂️⚔️🙍‍♂️"
	PartnerSingleManVsWoman = "🙍‍♂️⚔️🙍‍♀️"
	PartnerDoublesMan       = "🙍‍♂️🙍‍♂️⚔️🙍‍♂️🙍‍♂️"
	PartnerDoublesMixed     = "🙍‍♂️🙍‍♀️⚔️🙍‍♂️🙍‍♀️"
	PartnerSparring         = "🎯"
)

// Define constants for different sections of the single game process
const (
	PaymentHalf            = "💰 порівну"
	PaymentMeNothing       = "💰 я - 0%"
	PaymentMeAll           = "💰 я - 100%"
	PaymentPairs           = "💰 напополам парами"
	PaymentWhoLost         = "💰 програвший"
	PaymentPairsSeparately = "💰 кожний окремо"
)

// Define constants for different sections of the single game process
const (
	CourtWillSpecify = "🟩 корт: вкажу"
	CourtDoNotCare   = "🟩 корт: неважливо"
)

// Define constants for different sections of the single game process
const (
	DateToday       = "📅 сьогодні"
	DateTomorrow    = "📅 завтра"
	DateWillSpecify = "📅 вкажу"
)

// Define constants for different sections of the single game process
const (
	TimeDoNotCare   = "🕓 неважливо"
	TimeWillSpecify = "🕓 вкажу"
)

// Define constants for different sections of the single game process
const (
	callbackDataG                     = "a_0:0_0"
	callbackDataD1                    = "a_0:0_1"
	callbackDataD2                    = "a_0:0_2"
	callbackDataD3                    = "a_0:1_0"
	callbackDataO                     = "a_0:1_1"
	callbackDataP1                    = "a_0:1_2"
	callbackDataP2                    = "a_0:2_0"
	callbackDataS1                    = "a_0:2_1"
	callbackDataS2                    = "a_0:2_2"
	callbackDataSH                    = "a_0:3_0"
	callbackDataPartnerSingleManVsMan = "a_1:4_0"
	callbackPartnerSingleManVsWoman   = "b_1:4_0"
	callbackPartnerDoublesMan         = "c_1:4_0"
	callbackPartnerDoublesMixed       = "d_1:4_0"
	callbackPartnerSparring           = "e_1:4_0"
	callbackDateToday                 = "a_2:4_1"
	callbackDateTomorrow              = "b_2:4_1"
	callbackDateWillSpecify           = "c_2:4_1"
	callbackTimeDoNotCare             = "a_3:4_2"
	callbackTimeWillSpecify           = "b_3:4_2"
	callbackCourtWillSpecify          = "a_4:5_0"
	callbackCourtDoNotCare            = "b_4:5_0"
	callbackPaymentHalf               = "a_5:5_1"
	callbackPaymentMeNothing          = "b_5:5_1"
	callbackPaymentMeAll              = "c_5:5_1"
	callbackPaymentPairs              = "d_5:5_1"
	callbackPaymentWhoLost            = "e_5:5_1"
	callbackPaymentPairsSeparately    = "f_5:5_1"
)

// KeyboardCallback is a map of callback data to the corresponding text.
var KeyboardCallback = map[string]string{
	G:                     callbackDataG,
	D1:                    callbackDataD1,
	D2:                    callbackDataD2,
	D3:                    callbackDataD3,
	O:                     callbackDataO,
	P1:                    callbackDataP1,
	P2:                    callbackDataP2,
	S1:                    callbackDataS1,
	S2:                    callbackDataS2,
	SH:                    callbackDataSH,
	PartnerSingleManVsMan: callbackDataPartnerSingleManVsMan,
	DateToday:             callbackDateToday,
	TimeDoNotCare:         callbackTimeDoNotCare,
	CourtDoNotCare:        callbackCourtDoNotCare,
	PaymentHalf:           callbackPaymentHalf,
}

// KeyboardData is a map of text to the corresponding callback data.
var KeyboardData = map[string]string{
	callbackDataG:  G,
	callbackDataD1: D1,
	callbackDataD2: D2,
	callbackDataD3: D3,
	callbackDataO:  O,
	callbackDataP1: P1,
	callbackDataP2: P2,
	callbackDataS1: S1,
	callbackDataS2: S2,
	callbackDataSH: SH,
}

// TextInversionMap is a map of callback data to the corresponding text.
var TextInversionMap = map[string]string{
	callbackDataPartnerSingleManVsMan: PartnerSingleManVsWoman,
	callbackPartnerSingleManVsWoman:   PartnerDoublesMan,
	callbackPartnerDoublesMan:         PartnerDoublesMixed,
	callbackPartnerDoublesMixed:       PartnerSparring,
	callbackPartnerSparring:           PartnerSingleManVsMan,
	callbackDateToday:                 DateTomorrow,
	callbackDateTomorrow:              DateWillSpecify,
	callbackDateWillSpecify:           DateToday,
	callbackTimeDoNotCare:             TimeWillSpecify,
	callbackTimeWillSpecify:           TimeDoNotCare,
	callbackCourtWillSpecify:          CourtDoNotCare,
	callbackCourtDoNotCare:            CourtWillSpecify,
	callbackPaymentHalf:               PaymentMeNothing,
	callbackPaymentMeNothing:          PaymentMeAll,
	callbackPaymentMeAll:              PaymentPairs,
	callbackPaymentPairs:              PaymentWhoLost,
	callbackPaymentWhoLost:            PaymentPairsSeparately,
	callbackPaymentPairsSeparately:    PaymentHalf,
}

// CallbackInversionMap is a map of text to the corresponding callback data.
var CallbackInversionMap = map[string]string{
	callbackDataPartnerSingleManVsMan: callbackPartnerSingleManVsWoman,
	callbackPartnerSingleManVsWoman:   callbackPartnerDoublesMan,
	callbackPartnerDoublesMan:         callbackPartnerDoublesMixed,
	callbackPartnerDoublesMixed:       callbackPartnerSparring,
	callbackPartnerSparring:           callbackDataPartnerSingleManVsMan,
	callbackDateToday:                 callbackDateTomorrow,
	callbackDateTomorrow:              callbackDateWillSpecify,
	callbackDateWillSpecify:           callbackDateToday,
	callbackTimeDoNotCare:             callbackTimeWillSpecify,
	callbackTimeWillSpecify:           callbackTimeDoNotCare,
	callbackCourtWillSpecify:          callbackCourtDoNotCare,
	callbackCourtDoNotCare:            callbackCourtWillSpecify,
	callbackPaymentHalf:               callbackPaymentMeNothing,
	callbackPaymentMeNothing:          callbackPaymentMeAll,
	callbackPaymentMeAll:              callbackPaymentPairs,
	callbackPaymentPairs:              callbackPaymentWhoLost,
	callbackPaymentWhoLost:            callbackPaymentPairsSeparately,
	callbackPaymentPairsSeparately:    callbackPaymentHalf,
}

// Define constants for different sections of the single game process
const (
	Yes = "Так"
	No  = "Ні"
)

// Define constants for different sections of the single game process
const (
	GameConfirmationYes = "Підтверджую"
	GameConfirmationNo  = "Не підтверджую"
)

// Define constants for different sections of the single game process
var timeHour = [6][3]string{
	{"7:00", "13:00", "19:00"},
	{"8:00", "14:00", "20:00"},
	{"9:00", "15:00", "21:00"},
	{"10:00", "16:00", "22:00"},
	{"11:00", "17:00", "23:00"},
	{"12:00", "18:00", "0:00"},
}

// Define constants for different sections of the single game process
var timeHalfHour = [6][3]string{
	{"7:30", "13:30", "19:30"},
	{"8:30", "14:30", "20:30"},
	{"9:30", "15:30", "21:30"},
	{"10:30", "16:30", "22:30"},
	{"11:30", "17:30", "23:30"},
	{"12:30", "18:30", "0:30"},
}

// Define constants for different sections of the single game process
var (
	ProposeGameKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(G, KeyboardCallback[G]),
			tgbotapi.NewInlineKeyboardButtonData(D1, KeyboardCallback[D1]),
			tgbotapi.NewInlineKeyboardButtonData(D2, KeyboardCallback[D2]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(D3, KeyboardCallback[D3]),
			tgbotapi.NewInlineKeyboardButtonData(O, KeyboardCallback[O]),
			tgbotapi.NewInlineKeyboardButtonData(P1, KeyboardCallback[P1]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(P2, KeyboardCallback[P2]),
			tgbotapi.NewInlineKeyboardButtonData(S1, KeyboardCallback[S1]),
			tgbotapi.NewInlineKeyboardButtonData(S2, KeyboardCallback[S2]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(SH, KeyboardCallback[SH]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(PartnerSingleManVsMan, KeyboardCallback[PartnerSingleManVsMan]),
			tgbotapi.NewInlineKeyboardButtonData(DateToday, KeyboardCallback[DateToday]),
			tgbotapi.NewInlineKeyboardButtonData(TimeDoNotCare, KeyboardCallback[TimeDoNotCare]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(CourtDoNotCare, KeyboardCallback[CourtDoNotCare]),
			tgbotapi.NewInlineKeyboardButtonData(PaymentHalf, KeyboardCallback[PaymentHalf]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(Ok, Ok),
		),
	)

	ChoiceConfirmation = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(Yes, Yes),
			tgbotapi.NewInlineKeyboardButtonData(No, No),
		),
		// tgbotapi.NewInlineKeyboardRow(
		// 	tgbotapi.NewInlineKeyboardButtonData(Back, Back),
		// ),
	)
)

// NewTimeKeyboard is a function that returns a new time keyboard
func NewTimeKeyboard(halfHour bool) tgbotapi.InlineKeyboardMarkup {
	array := timeHour

	if halfHour {
		array = timeHalfHour
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(array[0][0], array[0][0]),
			tgbotapi.NewInlineKeyboardButtonData(array[0][1], array[0][1]),
			tgbotapi.NewInlineKeyboardButtonData(array[0][2], array[0][2]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(array[1][0], array[1][0]),
			tgbotapi.NewInlineKeyboardButtonData(array[1][1], array[1][1]),
			tgbotapi.NewInlineKeyboardButtonData(array[1][2], array[1][2]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(array[2][0], array[2][0]),
			tgbotapi.NewInlineKeyboardButtonData(array[2][1], array[2][1]),
			tgbotapi.NewInlineKeyboardButtonData(array[2][2], array[2][2]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(array[3][0], array[3][0]),
			tgbotapi.NewInlineKeyboardButtonData(array[3][1], array[3][1]),
			tgbotapi.NewInlineKeyboardButtonData(array[3][2], array[3][2]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(array[4][0], array[4][0]),
			tgbotapi.NewInlineKeyboardButtonData(array[4][1], array[4][1]),
			tgbotapi.NewInlineKeyboardButtonData(array[4][2], array[4][2]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(array[5][0], array[5][0]),
			tgbotapi.NewInlineKeyboardButtonData(array[5][1], array[5][1]),
			tgbotapi.NewInlineKeyboardButtonData(array[5][2], array[5][2]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(Back, Back),
		),
	)
}

// NewKeyboard is a function that returns a new keyboard
func NewKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(G, KeyboardCallback[G]),
			tgbotapi.NewInlineKeyboardButtonData(D1, KeyboardCallback[D1]),
			tgbotapi.NewInlineKeyboardButtonData(D2, KeyboardCallback[D2]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(D3, KeyboardCallback[D3]),
			tgbotapi.NewInlineKeyboardButtonData(O, KeyboardCallback[O]),
			tgbotapi.NewInlineKeyboardButtonData(P1, KeyboardCallback[P1]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(P2, KeyboardCallback[P2]),
			tgbotapi.NewInlineKeyboardButtonData(S1, KeyboardCallback[S1]),
			tgbotapi.NewInlineKeyboardButtonData(S2, KeyboardCallback[S2]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(SH, KeyboardCallback[SH]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(PartnerSingleManVsMan, KeyboardCallback[PartnerSingleManVsMan]),
			tgbotapi.NewInlineKeyboardButtonData(DateToday, KeyboardCallback[DateToday]),
			tgbotapi.NewInlineKeyboardButtonData(TimeDoNotCare, KeyboardCallback[TimeDoNotCare]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(CourtDoNotCare, KeyboardCallback[CourtDoNotCare]),
			tgbotapi.NewInlineKeyboardButtonData(PaymentHalf, KeyboardCallback[PaymentHalf]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(Ok, Ok),
			tgbotapi.NewInlineKeyboardButtonData(Back, Back),
		),
	)
}

// InitialMessage is a string that represents the initial message
const (
	InitialMessage = "Будь-ласка обирайте опціЇ. Треба тицнути на опцію."
)
