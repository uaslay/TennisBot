package ui

import (
    "os"
	"fmt"
	"math"
	"encoding/json"
)

const (
    GeneralRatingButton = "📊 Загальний рейтинг"
)

// Структура для збереження даних гравця
type Player struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Rating   int    `json:"rating"`
    Wins     int    `json:"wins"`
    Losses   int    `json:"losses"`
    Matches  int    `json:"matches"`
}

// Файл, де зберігаються гравці
const playersFile = "players.json"

// Завантажує гравців із файлу
func loadPlayers() map[string]Player {
    file, err := os.Open(playersFile)
    if err != nil {
        return make(map[string]Player) // Якщо файлу немає, повертаємо пусту мапу
    }
    defer file.Close()

    var players map[string]Player
    decoder := json.NewDecoder(file)
    err = decoder.Decode(&players)
    if err != nil {
        return make(map[string]Player)
    }
    return players
}

// Зберігає гравців у файл
func savePlayers(players map[string]Player) {
    file, err := os.Create(playersFile)
    if err != nil {
        panic(err) // Помилка запису у файл
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ") // Красиве форматування JSON
    err = encoder.Encode(players)
    if err != nil {
        panic(err)
    }
}

// Функція для визначення коефіцієнта K на основі рейтингу та матчів
func getKFactor(rating int, matches int) int {
	if matches < 30 {
		return 40
	} else if rating <= 600 {
		return 25
	} else if rating <= 2400 {
		return 20
	}
	return 10
}

// Функція для розрахунку очікуваного результату гравця A
func expectedScore(ratingA, ratingB int) float64 {
	return 1 / (1 + math.Pow(10, float64(ratingB-ratingA)/400))
}

// Функція для оновлення рейтингу після матчу
func UpdateElo(ratingA, ratingB, matchesA, matchesB int, resultA float64) (int, int) {
	E_A := expectedScore(ratingA, ratingB)
	E_B := expectedScore(ratingB, ratingA)

	K_A := getKFactor(ratingA, matchesA)
	K_B := getKFactor(ratingB, matchesB)

	newRatingA := ratingA + int(math.Round(float64(K_A) * (resultA - E_A)))
	newRatingB := ratingB + int(math.Round(float64(K_B) * ((1 - resultA) - E_B)))

	return newRatingA, newRatingB
}

// Оновлює рейтинг гравців після матчу
func updatePlayerRating(playerAID, playerBID string, resultA float64) {
    players := loadPlayers()

    // Якщо гравець A не існує – створюємо його
    playerA, existsA := players[playerAID]
    if !existsA {
        playerA = Player{
            ID:      playerAID,
            Name:    "Unknown",
            Rating:  0,
            Wins:    0,
            Losses:  0,
            Matches: 0,
        }
    }

    // Якщо гравець B не існує – створюємо його
    playerB, existsB := players[playerBID]
    if !existsB {
        playerB = Player{
            ID:      playerBID,
            Name:    "Unknown",
            Rating:  0,
            Wins:    0,
            Losses:  0,
            Matches: 0,
        }
    }

    // Оновлюємо рейтинги
    newRatingA, newRatingB := UpdateElo(playerA.Rating, playerB.Rating, playerA.Matches, playerB.Matches, resultA)

    playerA.Rating = newRatingA
    playerB.Rating = newRatingB
    playerA.Matches++
    playerB.Matches++

    players[playerAID] = playerA
    players[playerBID] = playerB

    savePlayers(players) // Зберігаємо оновлену базу
}


// Отримує рейтинг гравця разом із статистикою виграшів/поразок
func GetPlayerRating(playerID string) string {
    players := loadPlayers()

    player, exists := players[playerID]
    if !exists {
        // Якщо гравця немає – створюємо його з рейтингом 0
        player = Player{
            ID:      playerID,
            Name:    "Unknown",
            Rating:  0,  // Початковий рейтинг = 0
            Wins:    0,
            Losses:  0,
            Matches: 0,
        }
        players[playerID] = player
        savePlayers(players) // Зберігаємо нового гравця
    }

    return fmt.Sprintf("Ваш загальний рейтинг: %d (Виграш %d - %d Програш)", 
        player.Rating, player.Wins, player.Losses)
}


