package ui

import (
	"fmt"
    "log"
	"math"
    "errors" // Для повернення помилок
	"gorm.io/gorm"

    db "TennisBot/database"
)

const (
    GeneralRatingButton = "📊 Загальний рейтинг"
    FixScoreButton = "✍️ Зафіксувати рахунок"
)

// Функція для визначення коефіцієнта K на основі рейтингу та матчів
func GetKFactor(rating float64, matches int64) float64 {
	if matches < 30 {
		return 40.0
	} else if rating <= 600 { // Порівняння float64 з int - працює
		return 25.0
	} else if rating <= 2400 {
		return 20.0
	}
	return 10.0
}

// Функція для розрахунку очікуваного результату гравця A
func expectedScore(ratingA, ratingB float64) float64 {
	return 1.0 / (1.0 + math.Pow(10.0, (ratingB-ratingA)/400.0))
}

// Функція для оновлення рейтингу після матчу
func UpdateElo(ratingA, ratingB float64, matchesA, matchesB int64, resultA float64) (float64, float64) {
	E_A := expectedScore(ratingA, ratingB)
	E_B := expectedScore(ratingB, ratingA) // Очікуваний результат для B = 1 - E_A

	K_A := GetKFactor(ratingA, matchesA)
	K_B := GetKFactor(ratingB, matchesB)

	// Розрахунок нових рейтингів
	newRatingA := ratingA + K_A*(resultA-E_A)
	// Результат для B = 1 - resultA
	newRatingB := ratingB + K_B*((1.0-resultA)-E_B)

	// Округлення до найближчого цілого
	return math.Round(newRatingA), math.Round(newRatingB)
}

// UpdatePlayerRating оновлює рейтинг гравців після матчу, використовуючи базу даних
func UpdatePlayerRating(playerAID, playerBID int64, resultA float64, dbClient *db.DBClient) error {
	// 1. Отримати обох гравців з БД
	playerA, errA := dbClient.GetPlayer(playerAID)
	if errA != nil {
		log.Printf("Помилка отримання гравця A (ID: %d): %v", playerAID, errA)
		return fmt.Errorf("не вдалося отримати гравця A (ID: %d): %w", playerAID, errA)
	}

	playerB, errB := dbClient.GetPlayer(playerBID)
	if errB != nil {
		log.Printf("Помилка отримання гравця B (ID: %d): %v", playerBID, errB)
		return fmt.Errorf("не вдалося отримати гравця B (ID: %d): %w", playerBID, errB)
	}

	// 2. Розрахувати нові рейтинги
	newRatingA, newRatingB := UpdateElo(playerA.Rating, playerB.Rating, playerA.TotalMatches, playerB.TotalMatches, resultA)

	// 3. Оновити статистику для обох гравців
	playerA.Rating = newRatingA // Оновлюємо рейтинг
	playerB.Rating = newRatingB
	playerA.TotalMatches++      // Збільшуємо кількість матчів
	playerB.TotalMatches++

	if resultA == 1.0 { // Гравець A виграв
		playerA.Won++
		playerB.Lost++
	} else if resultA == 0.0 { // Гравець A програв (B виграв)
		playerA.Lost++
		playerB.Won++
	} else {
		// Можна обробити нічию (resultA == 0.5), якщо потрібно
		// У тенісі зазвичай нічиїх немає, але для інших ігор може бути актуально
		log.Printf("Нейтральний результат (%.1f) не змінює статистику перемог/програшів.", resultA)
	}

	// 4. Зберегти оновлені дані в БД
	errUpdateA := dbClient.UpdatePlayerStats(playerA) // Використовуємо нову функцію
	if errUpdateA != nil {
		log.Printf("Помилка оновлення статистики гравця A (ID: %d): %v", playerAID, errUpdateA)
		// Важливо: Що робити, якщо один гравець оновився, а інший ні? Потрібна транзакція!
		// Поки що просто повертаємо помилку.
		return fmt.Errorf("не вдалося оновити статистику гравця A: %w", errUpdateA)
	}

	errUpdateB := dbClient.UpdatePlayerStats(playerB)
	if errUpdateB != nil {
		log.Printf("Помилка оновлення статистики гравця B (ID: %d): %v", playerBID, errUpdateB)
		// Помилка другого оновлення, потенційна неузгодженість
		return fmt.Errorf("не вдалося оновити статистику гравця B: %w", errUpdateB)
	}

	log.Printf("Рейтинг оновлено: Гравець %d -> %.2f, Гравець %d -> %.2f", playerAID, newRatingA, playerBID, newRatingB)
	return nil // Успіх
}


// GetPlayerRating отримує рейтинг гравця з бази даних
func GetPlayerRating(playerID int64, dbClient *db.DBClient) string {
	player, err := dbClient.GetPlayer(playerID)
	if err != nil {
		// Гравець не знайдений або інша помилка БД
		log.Printf("Помилка отримання рейтингу для гравця ID %d: %v", playerID, err)
		// Можливо, варто повернути повідомлення про необхідність реєстрації
		return "Не вдалося отримати ваш рейтинг. Можливо, ви ще не зареєстровані? /start"
	}

	// Форматуємо рядок з даними гравця
	return fmt.Sprintf("Ваш загальний рейтинг: %.0f (Виграш %d - %d Програш, Матчів: %d)",
		player.Rating, // Використовуємо %.0f для відображення як ціле
		player.Won,
		player.Lost,
		player.TotalMatches)
}

// GetPlayerByUsername шукає гравця за юзернеймом у базі даних
func GetPlayerByUsername(username string, dbClient *db.DBClient) (int64, bool) {
	player, err := dbClient.GetPlayerByUsername(username)
	if err != nil {
		// Помилка (включаючи не знайдено)
		if !errors.Is(err, gorm.ErrRecordNotFound) { // Логуємо тільки "справжні" помилки БД
			log.Printf("Помилка пошуку гравця за UserName '%s': %v", username, err)
		}
		return 0, false // Гравець не знайдений або помилка
	}
	// Гравець знайдений
	return player.UserID, true
}
