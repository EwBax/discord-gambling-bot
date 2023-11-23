// This file acts as a database adapter and handles interactions with the database for this program.
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// DbPath The connection string for the database.
// Because it is just a sqlite database stored in the same directory as the program, it is okay to have this hard coded.
const DbPath string = "casino.db"

type DBA struct {
	conn *sql.DB
}

// OpenConnection Opens the connection to a sqlite3 database
func (dba *DBA) OpenConnection(connectionString string) {

	//Opening connection to database

	var err error
	dba.conn, err = sql.Open("sqlite3", connectionString)

	// If connection cannot be made, program should shut down.
	if err != nil {
		log.Fatal(err)
	}

}

// FindPlayer Queries the database for a player and returns their info. OR creates a new player if player cannot be found
func (dba *DBA) FindPlayer(username string) Player {

	row := dba.conn.QueryRow(fmt.Sprintf("SELECT * FROM player WHERE username='%s'", username))

	// Creating the player and scanning the info into it
	var player Player
	err := row.Scan(&player.ID, &player.Username, &player.Chips, &player.Wins, &player.Ties, &player.Losses)

	// If no rows are returned the player has to be created
	if errors.Is(err, sql.ErrNoRows) {
		return dba.CreatePlayer(username)
	}

	return player

}

// CreatePlayer Creates a new entry in the database for the player, and returns
func (dba *DBA) CreatePlayer(username string) Player {

	// Creating the new player object. All not included parameter names are integers and will be set to their zero value (which is 0)
	newPlayer := Player{Username: username, Chips: StartingChips}
	// Not inserting the wins, losses, or ties because those default to 0 in the database
	res, err := dba.conn.Exec(fmt.Sprintf("INSERT INTO player VALUES(NULL,'%s', '%d', '%d', '%d', '%d');", newPlayer.Username, newPlayer.Chips, newPlayer.Wins, newPlayer.Ties, newPlayer.Losses))
	if err != nil {
		log.Fatal(err)
	}

	// Getting the ID of the player we just created
	var id int64
	id, err = res.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	newPlayer.ID = int(id)

	return newPlayer

}

// UpdatePlayer updates the entry for the player that is passed to the player's new stats (chip total, wins, losses)
func (dba *DBA) UpdatePlayer(player Player) {

	_, err := (*dba).conn.Exec(
		fmt.Sprintf("UPDATE player SET chips='%d', wins='%d', ties='%d', losses='%d' WHERE id='%d'",
			player.Chips,
			player.Wins,
			player.Ties,
			player.Losses,
			player.ID))

	if err != nil {
		log.Fatal(err)
	}

}

// GetChipTotal queries the database for a player using username, and returns their chip total.
func (dba *DBA) GetChipTotal(username string) int {

	row := dba.conn.QueryRow(fmt.Sprintf("SELECT chips FROM player WHERE username='%s'", username))

	var chipTotal int
	err := row.Scan(&chipTotal)

	// If no rows are returned the player has to be created
	if errors.Is(err, sql.ErrNoRows) {
		chipTotal = dba.CreatePlayer(username).Chips
	}

	return chipTotal

}

const (
	Wins  = 0
	Chips = 1
)

// GetLeaderboard queries the database for all players ordered by wins descending.
// leaderboardType = 0 for sorting by wins, 1 for sorting by chips.
func (dba *DBA) GetLeaderboard(leaderboardType int) []Player {

	var temp string

	switch leaderboardType {
	case Wins:
		temp = "wins"
	case Chips:
		temp = "chips"
	default:
		panic("Invalid leaderboard_type")
	}

	// Querying the database
	rows, err := dba.conn.Query(fmt.Sprintf("SELECT * FROM player ORDER BY player.%s DESC, player.Username ASC;", temp))

	if err != nil {
		log.Fatal(err)
	}

	var leaderboard []Player
	player := Player{}

	// appending each row to the leaderboard slice
	for rows.Next() {

		err = rows.Scan(&player.ID, &player.Username, &player.Chips, &player.Wins, &player.Ties, &player.Losses)

		if err != nil {
			log.Fatal(err)
		}

		leaderboard = append(leaderboard, player)
	}

	return leaderboard

}
