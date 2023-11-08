package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Global variable for database connection and connection string
var db *sql.DB

const db_path string = "./db/casino.db"

// Whether the bot is currently playing a game with someone
var GAME_ON = false

// Global variable for a game of blackjack
var BlackjackGame Blackjack

func main() {

	// Opening connection to database
	// db, err := sql.Open("sqlite3", db_path)

	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Creating a new Discord session using the bot token
	sess, err := discordgo.New("Bot MTE1NzAzOTk4NTgxNjUxNDcwMQ.GOVQIL.KyV-XFZB0f1ylsYC0PQqOwFH5yw5vwgwWVupBM")
	if err != nil {
		log.Fatal(err)
	}

	// Handles a message being created in discord
	sess.AddHandler(MessageRecieved)

	// Sets the intent for the session
	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = sess.Open()

	if err != nil {
		log.Fatal(err)
	}

	defer sess.Close()

	fmt.Println("The bot is online.")

	// This makes the bot continue to run
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func MessageRecieved(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Checks that the author was a user other than the bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	if GAME_ON && m.Author.Username == BlackjackGame.PlayerName {

		if BlackjackGame.IsPlayersTurn {
			if strings.ToLower(m.Content) == "!hit" {
				s.ChannelMessageSend(m.ChannelID, "You chose to hit!")
				BlackjackGame.Hit(&BlackjackGame.PlayerHand)
				BlackjackGame.RunPlayerTurn(s, m)
			} else if strings.ToLower(m.Content) == "!hold" {
				s.ChannelMessageSend(m.ChannelID, "You chose to hold!")
				BlackjackGame.IsPlayersTurn = false
				BlackjackGame.RunDealerTurn(s, m)
			}
		}

	} else {

		// If a user enters !blackjack we begin the game
		if strings.ToLower(m.Content) == "!blackjack" {
			GAME_ON = true
			BlackjackGame = NewBlackjack(m.Author.Username)
			s.ChannelMessageSend(m.ChannelID, "Starting a new game of blackjack with "+m.Author.Username+"!")
			BlackjackGame.RunPlayerTurn(s, m)
		}

	}

}

func GameOver(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Game over!")
	GAME_ON = false
}

// func FindPlayer(username string) bool {

// 	row := db.QueryRow("SELECT * FROM Player WHERE username=?;", username)

// 	return row
// }
