package database

// TODO: protect with mutex or else

// CreatePlayer creates a player in the database
func (dbClient *DBClient) CreatePlayer(player Player) {
	dbClient.DB.Create(&player)
}

// CheckPlayerRegistration checks if a player is registered in the database
func (dbClient DBClient) CheckPlayerRegistration(UserID int64) bool {
	var player Player

	dbClient.DB.Where("user_id = ?", UserID).First(&player)

	return player.UserID != 0
}

// GetPlayer returns a player by UserID
func (dbClient DBClient) GetPlayer(UserID int64) (player Player) {
	var user Player

	dbClient.DB.Where("user_id = ?", UserID).First(&user)

	return user
}

// UpdatePlayer updates a player's racket in the database
func (dbClient DBClient) UpdatePlayer(value string, userID int64) {
	dbClient.DB.Model(&Player{}).Where("user_id = ?", userID).Update("Racket", value)
}

// CreateGame creates a game in the database
func (dbClient DBClient) CreateGame(game ProposedGame) {
	dbClient.DB.Create(&game)
}

// GetGame returns a game by GameID
func (dbClient DBClient) GetGame(GameID uint) ProposedGame {
	var game ProposedGame

	dbClient.DB.Where("ID = ?", GameID).First(&game)

	return game

}

// GetGameID returns a game's UserID by GameID
func (dbClient DBClient) GetGameID(GameID uint) int64 {
	var game ProposedGame

	dbClient.DB.Where("ID = ?", GameID).First(&game)

	return game.UserID

}

// DeleteGame deletes a game by GameID
func (dbClient DBClient) DeleteGame(ID uint) {
	dbClient.DB.Delete(&ProposedGame{}, ID)
}

// GetGames returns all games in the database
func (dbClient DBClient) GetGames() (userGames []ProposedGame) {
	var games []ProposedGame

	dbClient.DB.Order("date asc").Find(&games)

	return games
}

// GetGamesByUserID returns all games by UserID
func (dbClient DBClient) GetGamesByUserID(UserID int64) (userGames []ProposedGame) {
	var games []ProposedGame

	dbClient.DB.Where("user_id = ?", UserID).Find(&games).Order("date asc")

	return games
}
