// File: database/crud.go
package database

import (
	"fmt"
	"errors" // Для повернення помилок
	"gorm.io/gorm"
)

// TODO: protect with mutex or else (Зауваження: GORM сам по собі обробляє конкурентність на рівні підключення до БД, але для складних транзакцій може знадобитися додатковий контроль)

// CreatePlayer creates a player in the database
func (dbClient *DBClient) CreatePlayer(player Player) error {
	result := dbClient.DB.Create(&player)
	return result.Error
}

// CheckPlayerRegistration checks if a player is registered in the database
func (dbClient DBClient) CheckPlayerRegistration(UserID int64) bool {
	var player Player
	// Використовуємо First, він поверне помилку gorm.ErrRecordNotFound, якщо запис не знайдено
	err := dbClient.DB.Where("user_id = ?", UserID).First(&player).Error
	// Повертаємо true, якщо помилки НЕМАЄ (тобто запис знайдено)
	return err == nil
}

// GetPlayer returns a player by UserID
func (dbClient DBClient) GetPlayer(UserID int64) (player Player, err error) {
	// First повертає помилку, якщо гравець не знайдений
	result := dbClient.DB.Where("user_id = ?", UserID).First(&player)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return player, fmt.Errorf("гравця з ID %d не знайдено", UserID)
		}
		return player, result.Error // Інша помилка бази даних
	}
	return player, nil
}

// GetPlayerByUsername returns a player by UserName
func (dbClient DBClient) GetPlayerByUsername(username string) (player Player, err error) {
	// Додаємо @, якщо його немає, для уніфікації пошуку
	if len(username) > 0 && username[0] != '@' {
		username = "@" + username
	}
	// First повертає помилку, якщо гравець не знайдений
	result := dbClient.DB.Where("user_name = ?", username).First(&player)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return player, fmt.Errorf("гравця з UserName %s не знайдено", username)
		}
		return player, result.Error // Інша помилка бази даних
	}
	return player, nil
}

// UpdatePlayer updates specific fields for a player by UserID
// Example usage: dbClient.UpdatePlayer(userID, map[string]interface{}{"Racket": "New Racket", "Rating": 1550.5})
func (dbClient DBClient) UpdatePlayer(userID int64, updates map[string]interface{}) error {
	result := dbClient.DB.Model(&Player{}).Where("user_id = ?", userID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	// Перевіряємо, чи було оновлено хоча б один запис
	if result.RowsAffected == 0 {
		// Це може статися, якщо гравця з таким userID не існує
		// Можна повернути помилку або просто проігнорувати, залежно від логіки
		return fmt.Errorf("гравця з ID %d не знайдено для оновлення", userID)
	}
	return nil
}

// UpdatePlayerStats updates rating and statistics for a player
// Простіший варіант, якщо оновлюємо всі поля статистики разом
func (dbClient DBClient) UpdatePlayerStats(player Player) error {
	result := dbClient.DB.Model(&Player{}).Where("user_id = ?", player.UserID).Updates(map[string]interface{}{
		"Rating":       player.Rating,
		"Won":          player.Won,
		"Lost":         player.Lost,
		"TotalMatches": player.TotalMatches,
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("гравця з ID %d не знайдено для оновлення статистики", player.UserID)
	}
	return nil
}

// --- Functions for ProposedGame (залишаємо або адаптуємо за потреби) ---

// CreateGame creates a game in the database
func (dbClient DBClient) CreateGame(game ProposedGame) error {
	result := dbClient.DB.Create(&game)
	return result.Error
}

// GetGame returns a game by GameID
func (dbClient DBClient) GetGame(GameID uint) (ProposedGame, error) {
	var game ProposedGame
	result := dbClient.DB.First(&game, GameID) // Пошук за первинним ключем ID
	if result.Error != nil {
		return game, result.Error
	}
	return game, nil
}

// GetGameID returns a game's UserID by GameID
// Ця функція виглядає менш потрібною, якщо GetGame повертає всю гру
// Можливо, варто переглянути її використання
func (dbClient DBClient) GetGameID(GameID uint) (int64, error) {
	var game ProposedGame
	// Отримуємо тільки поле user_id
	result := dbClient.DB.Model(&ProposedGame{}).Select("user_id").First(&game, GameID)
	if result.Error != nil {
		return 0, result.Error
	}
	return game.UserID, nil
}

// DeleteGame deletes a game by GameID
func (dbClient DBClient) DeleteGame(ID uint) error {
	result := dbClient.DB.Delete(&ProposedGame{}, ID)
	return result.Error
}

// GetGames returns all games in the database
func (dbClient DBClient) GetGames() ([]ProposedGame, error) {
	var games []ProposedGame
	result := dbClient.DB.Order("created_at desc").Find(&games) // Можливо, краще сортувати за датою створення
	return games, result.Error
}

// GetGamesByUserID returns all games by UserID
func (dbClient DBClient) GetGamesByUserID(UserID int64) ([]ProposedGame, error) {
	var games []ProposedGame
	result := dbClient.DB.Where("user_id = ?", UserID).Order("created_at desc").Find(&games)
	return games, result.Error
}