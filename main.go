package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// GAME_ON Whether the bot is currently playing a game with someone
var GAME_ON = false

// BlackjackGame Global variable for a game of blackjack
var BlackjackGame Blackjack

// The database adapter
var dba DBA

func main() {

	// Opening the database connection
	dba.OpenConnection(DbPath)

	// Creating a new Discord session using the bot token
	sess, err := discordgo.New("Bot MTE1NzAzOTk4NTgxNjUxNDcwMQ.GOVQIL.KyV-XFZB0f1ylsYC0PQqOwFH5yw5vwgwWVupBM")
	if err != nil {
		log.Fatal(err)
	}

	// Handles a message being created in discord
	sess.AddHandler(MessageReceived)

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

func MessageReceived(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Checks that the author was a user other than the bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	if GAME_ON {

		if m.Author.Username == BlackjackGame.Player.Username {

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
		}

	} else {

		// If a user enters !blackjack we begin the game
		if strings.ToLower(m.Content) == "!blackjack" {
			GAME_ON = true
			player := dba.FindPlayer(m.Author.Username)
			BlackjackGame = NewBlackjack(player)
			s.ChannelMessageSend(m.ChannelID, "Starting a new game of blackjack with "+m.Author.Username+"!")
			BlackjackGame.RunPlayerTurn(s, m)
		}

	}

}

func GameOver(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Game over!")
	GAME_ON = false
}
