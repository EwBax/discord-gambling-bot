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
	err := row.Scan(&player.ID, &player.Username, &player.Chips)

	// If no rows are returned the player has to be created
	if errors.Is(err, sql.ErrNoRows) {
		return dba.CreatePlayer(username)
	}

	return player

}

// CreatePlayer Creates a new entry in the database for the player, and returns
func (dba *DBA) CreatePlayer(username string) Player {

	// Creating the new player object
	newPlayer := Player{0, username, StartingChips}
	res, err := dba.conn.Exec(fmt.Sprintf("INSERT INTO player VALUES(NULL,'%s','%d');", newPlayer.Username, newPlayer.Chips))
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

func (dba *DBA) UpdateChipBalance(player Player) {

	_, err := (*dba).conn.Exec(fmt.Sprintf("UPDATE player SET chip_total='%d' WHERE id='%d'", player.Chips, player.ID))
	if err != nil {
		log.Fatal(err)
	}

}
