// File: database/crud.go
package database

import (
	"log"
	"fmt"
	"errors" // Для повернення помилок
	"gorm.io/gorm"
)

// TODO: protect with mutex or else (Зауваження: GORM сам по собі обробляє конкурентність на рівні підключення до БД, але для складних транзакцій може знадобитися додатковий контроль)

// --- Функції для Player ---

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
		// Повертаємо помилку, щоб бачити, якщо щось не так
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

// --- Функції для ProposedGame ---

// CreateGame creates a game in the database
func (dbClient DBClient) CreateGame(game ProposedGame) error {
	result := dbClient.DB.Create(&game)
	return result.Error
}

// GetGame returns all games in the database, preloading Player data
func (dbClient DBClient) GetGame(ID uint) ([]ProposedGame, error) {
	var games []ProposedGame
	// ---> ДОДАЄМО .Preload("Player") <---
	result := dbClient.DB.Preload("Player").Order("created_at desc").Find(&games)
	// ---> Кінець зміни <---
	if result.Error != nil {
		log.Printf("Помилка отримання списку ігор з Preload: %v", result.Error)
	}
	return games, result.Error
}

// DeleteGame deletes a game by GameID
// ВАЖЛИВО: Ця функція видаляє ТІЛЬКИ ProposedGame.
// Асоційовані GameResponse мають видалятися окремо або каскадно (якщо налаштовано в БД/GORM).
// Для безпеки краще видаляти GameResponse явно перед викликом DeleteGame або в транзакції.
func (dbClient DBClient) DeleteGame(ID uint) error {
	result := dbClient.DB.Delete(&ProposedGame{}, ID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Не обов'язково помилка, гра могла бути видалена раніше
		return gorm.ErrRecordNotFound // Повертаємо стандартну помилку GORM
	}
	return result.Error
}

// // GetGames returns all games in the database (consider adding filters, e.g., only future games)
// func (dbClient DBClient) GetGames() ([]ProposedGame, error) {
// 	var games []ProposedGame
// 	result := dbClient.DB.Order("created_at desc").Find(&games) // Можливо, краще сортувати за датою гри?
// 	return games, result.Error
// }

// GetGamesByUserID returns all games by UserID
func (dbClient DBClient) GetGamesByUserID(userID int64) ([]ProposedGame, error) {
    var player Player
    // Спочатку знаходимо гравця за його Telegram ID (user_id)
    err := dbClient.DB.Where("user_id = ?", userID).First(&player).Error
    if err != nil {
        // Обробляємо помилку: гравець не знайдений або інша помилка БД
        return nil, fmt.Errorf("не вдалося знайти гравця з user_id %d: %w", userID, err)
    }

    // Тепер шукаємо ігри, використовуючи первинний ключ гравця (player.ID uint)
    var games []ProposedGame
    // Припускаємо, що ProposedGame тепер має поле PlayerID uint
    result := dbClient.DB.Where("player_id = ?", player.ID).Order("created_at desc").Find(&games)
    return games, result.Error
}

// --- Функції для GameResponse ---

// CreateGameResponse creates a response in the database
func (dbClient *DBClient) CreateGameResponse(response GameResponse) error {
	result := dbClient.DB.Create(&response)
	return result.Error
}

// GetGameResponsesByGameID returns all responses for a specific game ID
func (dbClient DBClient) GetGameResponsesByGameID(gameID uint) ([]GameResponse, error) {
	var responses []GameResponse
	// Використовуємо Preload для завантаження даних Responder (Player)
	result := dbClient.DB.Preload("Responder").Where("proposed_game_id = ?", gameID).Find(&responses)
	return responses, result.Error
}

// DeleteGameResponsesByGameID deletes all responses associated with a specific game ID
// Повертає кількість видалених записів та помилку
func (dbClient DBClient) DeleteGameResponsesByGameID(gameID uint) (int64, error) {
	result := dbClient.DB.Where("proposed_game_id = ?", gameID).Delete(&GameResponse{})
	return result.RowsAffected, result.Error
}

// DeleteGameResponseByID deletes a specific response by its ID
func (dbClient DBClient) DeleteGameResponseByID(responseID uint) error {
	result := dbClient.DB.Delete(&GameResponse{}, responseID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CheckExistingResponse checks if a specific player has already responded to a specific game
func (dbClient DBClient) CheckExistingResponse(gameID uint, responderID int64) (bool, error) {
	var count int64
	result := dbClient.DB.Model(&GameResponse{}).Where("proposed_game_id = ? AND responder_id = ?", gameID, responderID).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// --- Функції для DualGame ---

// (Залишаємо без змін, якщо вони ще потрібні)
