// Package database package contains all the structs that represent the tables in the database
package database

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Player is a struct that represents a player in the database
type Player struct {
	UserID                          int64
	NameSurname                     string
	YearOfBirth                     int
	YearStartedPlaying              int
	YearsOfPlayingWithoutInterrupts int
	PlayedGames                     int64
	ChampionshipsParticipation      bool
	City                            string
	Area                            string
	Rating                          int16
	Racket                          string
	Won                             int32
	Lost                            int32
	AvatarPhotoPath                 string
	MobileNumber                    string
	UserName                        string
	FavouriteCourt                  string
	MainHand                        string
	gorm.Model
}

// ProposedGame is a struct that represents a proposed game in the database
type ProposedGame struct {
	UserID        int64
	RegionSection string
	Partner       string
	Date          string
	Time          string
	Court         string
	Payment       string
	gorm.Model
}

// String returns a string representation of a player
func (p Player) String() string {
	return fmt.Sprintf(
		"%s\n%s : %s\nРейтинг: %d\nРакетка: %s\nЗіграв матчів: %d, з яких:\nВиграв: %d\nПрограв: %d\nУлюблене покриття: %s\nІгрова рука: %s\nТелефон: %s\nUserName: %s",
		p.NameSurname,
		p.City,
		p.Area,
		p.Rating,
		p.Racket,
		p.PlayedGames,
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

// String returns a string representation of a proposed game
func (g ProposedGame) String() string {
	unixTimestamp, _ := strconv.ParseInt(g.Date, 10, 64)
	unixTime := time.Unix(unixTimestamp, 0)
	date := ConvertDayToUkr(int(unixTime.Weekday())) + " " + strconv.Itoa(unixTime.Day())

	var b strings.Builder

	fmt.Fprintf(&b, "%s, ", date)
	fmt.Fprintf(&b, "%s, ", g.Time)
	fmt.Fprintf(&b, "%s, ", g.Partner)
	fmt.Fprintf(&b, "%s", g.Payment)

	if g.RegionSection != "" {
		fmt.Fprintf(&b, ", %s", g.RegionSection)
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
