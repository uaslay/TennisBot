package main

import (
    "os"
	"math"
	"encoding/json"
)

// Структура профілю гравця
type Player struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Rating   int    `json:"rating"`
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
func updateElo(ratingA, ratingB, matchesA, matchesB int, resultA float64) (int, int) {
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

    playerA, existsA := players[playerAID]
    playerB, existsB := players[playerBID]

    if !existsA || !existsB {
        return // Гравці не знайдені
    }

    // Оновлюємо рейтинги
    newRatingA, newRatingB := updateElo(playerA.Rating, playerB.Rating, playerA.Matches, playerB.Matches, resultA)

    playerA.Rating = newRatingA
    playerB.Rating = newRatingB
    playerA.Matches++
    playerB.Matches++

    players[playerAID] = playerA
    players[playerBID] = playerB

    savePlayers(players)
}

// Отримує рейтинг гравця за його ID
func getPlayerRating(playerID string) string {
    players := loadPlayers()

    player, exists := players[playerID]
    if !exists {
        return "Гравець не знайдений. Можливо, ви ще не зареєстровані."
    }

    return fmt.Sprintf("Гравець: %s\nРейтинг: %d\nМатчів зіграно: %d", player.Name, player.Rating, player.Matches)
}
