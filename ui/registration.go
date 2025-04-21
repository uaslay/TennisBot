package ui

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// List of input buttons
const (
	ProfileButton = "👤 Мій профіль"
	SingleGame    = "🎾 Разова гра"
	GameProposal  = "Відповідь"
)

// List of input options
const (
	ProfileRegistrationStart                    = "Для користування ботом просимо пройти швидку реєстрацію."
	ProfileMsgNameSurname                       = "Будь-ласка введіть ім'я та прізвище."
	ProfileMsgCity                              = "Місто ?"
	ProfileMsgArea                              = "Район міста ?"
	ProfileMsgAge                               = "Рік народження ?"
	ProfileMsgFromWhatAge                       = "З якого віку займаєтесь тенісом ?"
	ProfileMsgHowManyYearsInTennis              = "Скільки років займаєтесь тенісом без урахування перерв ?"
	ProfileMsgHowManyParticipationInTournaments = "Чи приймали Ви участь у змаганнях (аматорських або професійних) ?"
	ProfileFavouriteCourtMaterial               = "Улюблено покриття ?"
	ProfileMainHand                             = "Правша чи лівша ?"
	ProfileMsgPhoto                             = "Можете додати Фотографію/"
	ProfileMsgSuccessfulRegistration            = "Реєстрація пройшла успішно."
)

// ProfileRegistrationSteps is a type for registration steps
type ProfileRegistrationSteps int16

// List of registration steps
const (
	NameSurname ProfileRegistrationSteps = iota
	Age
	FromWhatAge
	HowManyYears
	ChampionshipsParticipation
	RegistrationFinished
	City
	Area
	MobileNumber
	FavouriteCourtMaterial
	MainHand
)

// List of input options
const (
	G  = "Гол"
	D1 = "Дар"
	D2 = "Дес"
	D3 = "Днi"
	O  = "Обо"
	P1 = "Печ"
	P2 = "Под"
	S1 = "Свя"
	S2 = "Сол"
	SH = "Шев"
)

// List of input options
const (
	Hard   = "Хард"
	Ground = "Грунт"
	Grass  = "Трава"
)

// List of input options
const (
	Right = "Права"
	Left  = "Ліва"
)

// List input options
var (
	DefaultAvatarFileID = "/resources/defaultAvatar.jpg"

	Championships = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Так", "Так"),
			tgbotapi.NewInlineKeyboardButtonData("Ні", "Ні"),
		),
	)

	KyivRegions = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(G, G),
			tgbotapi.NewInlineKeyboardButtonData(D1, D1),
			tgbotapi.NewInlineKeyboardButtonData(D2, D2),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(D3, D3),
			tgbotapi.NewInlineKeyboardButtonData(O, O),
			tgbotapi.NewInlineKeyboardButtonData(P1, P1),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(P2, P2),
			tgbotapi.NewInlineKeyboardButtonData(S1, S1),
			tgbotapi.NewInlineKeyboardButtonData(S2, S2),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(SH, SH),
		),
	)

	FavourityCourtMaterial = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(Hard, Hard),
			tgbotapi.NewInlineKeyboardButtonData(Ground, Ground),
			tgbotapi.NewInlineKeyboardButtonData(Grass, Grass),
		),
	)

	MainGameHand = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(Right, Right),
			tgbotapi.NewInlineKeyboardButtonData(Left, Left),
		),
	)
)
