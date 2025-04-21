// Package database package contains all the structs that represent the tables in the database
package database

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Player is a struct that represents a player in the database
type Player struct {
	UserID                          int64 `gorm:"uniqueIndex"`
	NameSurname                     string
	YearOfBirth                     int
	YearStartedPlaying              int
	YearsOfPlayingWithoutInterrupts int
	TotalMatches                    int64
	ChampionshipsParticipation      bool
	City                            string
	Area                            string
	Rating                          float64
	Racket                          string
	Won                             int64
	Lost                            int64
	AvatarFileID                    string
	MobileNumber                    string
	UserName                        string `gorm:"uniqueIndex"`
	FavouriteCourt                  string
	MainHand                        string
	ProposedGames                   []ProposedGame `gorm:"foreignKey:PlayerID"`      // Зв'язок один-до-багатьох
	GameResponses                   []GameResponse `gorm:"foreignKey:ResponderID"` // Зв'язок один-до-багатьох
	gorm.Model
}

// ProposedGame is a struct that represents a proposed game in the database
type ProposedGame struct {
	PlayerID      uint
	RegionSection string
	Partner       string
	Date          string
	Time          string
	Court         string
	Payment       string
	Player        Player         `gorm:"foreignKey:PlayerID"`         // Зв'язок багато-до-одного
	GameResponses []GameResponse `gorm:"foreignKey:ProposedGameID"` // Зв'язок один-до-багатьох
	gorm.Model
}

// GameResponse is a struct that represents a response to a proposed game
type GameResponse struct {
	ProposedGameID uint        
	ResponderID    uint        
	ProposedGame   ProposedGame `gorm:"foreignKey:ProposedGameID"` // Зв'язок багато-до-одного
	Responder      Player       `gorm:"foreignKey:ResponderID"`    // Зв'язок багато-до-одного
	gorm.Model                  // Includes ID, CreatedAt, UpdatedAt, DeletedAt
}

// String returns a string representation of a player
func (p Player) String() string {
	return fmt.Sprintf(
		"%s\n%s : %s\nРейтинг: %.0f\nРакетка: %s\nЗіграв матчів: %d, з яких:\nВиграв: %d\nПрограв: %d\nУлюблене покриття: %s\nІгрова рука: %s\nТелефон: %s\nUserName: %s",
		p.NameSurname,
		p.City,
		p.Area,
		p.Rating,
		p.Racket,
		p.TotalMatches,
		p.Won,
		p.Lost,
		p.FavouriteCourt,
		p.MainHand,
		p.MobileNumber,
		p.UserName,
	)
}

// ConvertDayToUkr converts a day to Ukrainian
func ConvertDayToUkr(day int) string {
	wd := time.Weekday(day) // Конвертуємо int в time.Weekday
	switch wd {
	case time.Monday:
		return "Пн"
	case time.Tuesday:
		return "Вт"
	case time.Wednesday:
		return "Ср"
	case time.Thursday:
		return "Чт"
	case time.Friday:
		return "Пт"
	case time.Saturday:
		return "Сб"
	case time.Sunday:
		return "Нд"
	default:
		return ""
	}
}

// String returns a string representation of a proposed game
func (g ProposedGame) String() string {
	unixTimestamp, err := strconv.ParseInt(g.Date, 10, 64)
	dateStr := g.Date // Fallback
	if err == nil {
		unixTime := time.Unix(unixTimestamp, 0)
		dateStr = ConvertDayToUkr(int(unixTime.Weekday())) + " " + strconv.Itoa(unixTime.Day())
	} else {
		log.Printf("Error parsing ProposedGame.Date ('%s') as Unix timestamp for game ID %d: %v", g.Date, g.ID, err)
	}

	var b strings.Builder

	fmt.Fprintf(&b, "%s, ", dateStr)
	fmt.Fprintf(&b, "%s, ", g.Time)
	fmt.Fprintf(&b, "%s, ", g.Partner)
	fmt.Fprintf(&b, "%s", g.Payment)

	if g.RegionSection != "" && g.RegionSection != "Не вказано" { // Додано перевірку на "Не вказано"
		fmt.Fprintf(&b, ", %s", g.RegionSection)
	}
	if g.Court != "" && g.Court != "Неважливо" { // Додано вивід корту, якщо вказано
		fmt.Fprintf(&b, ", Корт: %s", g.Court)
	}

	return b.String()
}

// https://stackoverflow.com/questions/63256680/adding-an-array-of-integers-as-a-data-type-in-a-gorm-model

// DualGame is a struct that represents a dual game in the database
type DualGame struct {
	ProposedPlayerID      int64
	RespondedPlayerID     int64
	ConfirmationProposed  bool
	ConfirmationResponded bool
	Score                 pq.StringArray `gorm:"type:text[]"`
	BothConfirmed         bool
	gorm.Model
}
