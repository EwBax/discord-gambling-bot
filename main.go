package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// GameOn Whether the bot is currently playing a game with someone
var GameOn = false

// BlackjackGame Global variable for a game of blackjack
var BlackjackGame Blackjack

// The database adapter
var dba DBA

func main() {

	config := GetConfig()

	// Opening the database connection
	dba.OpenConnection(DbPath)

	// Creating a new Discord session using the bot token
	sess, err := discordgo.New(config.Token)
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

	if strings.ToLower(m.Content) == "!balance" {
		chipTotal := dba.GetChipTotal(m.Author.Username)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, your chip total is: %d", m.Author.Username, chipTotal))
	} else if strings.ToLower(m.Content) == "!leaderboard wins" {
		player := dba.FindPlayer(m.Author.Username)
		DisplayLeaderboard(player, "wins", s, m)
	} else if strings.ToLower(m.Content) == "!leaderboard chips" {
		player := dba.FindPlayer(m.Author.Username)
		DisplayLeaderboard(player, "chips", s, m)
	} else if strings.ToLower(m.Content) == "!leaderboard" {
		s.ChannelMessageSend(m.ChannelID, "You can do !leaderboard wins or !leaderboard chips for a leaderboard sorted for each statistic!")
	} else if GameOn {

		if m.Author.Username == BlackjackGame.Player.Username {

			if BlackjackGame.IsPlayersTurn {
				if strings.ToLower(m.Content) == "!hit" {
					s.ChannelMessageSend(m.ChannelID, "You chose to hit!")
					BlackjackGame.Hit(&BlackjackGame.PlayerHand)
					BlackjackGame.RunPlayerTurn(s, m)
				} else if strings.ToLower(m.Content) == "!stand" {
					s.ChannelMessageSend(m.ChannelID, "You chose to stand!")
					BlackjackGame.IsPlayersTurn = false
					BlackjackGame.RunDealerTurn(s, m)
				}
			}
		}

	} else {

		blackjackRegex := regexp.MustCompile(`!blackjack -?\d+`)

		// If a user enters !blackjack and a bet amount, we begin the game
		if blackjackRegex.MatchString(strings.ToLower(m.Content)) {

			// Getting the wager amount from the !blackjack command
			wager, _ := strconv.Atoi(strings.Split(m.Content, " ")[1])
			player := dba.FindPlayer(m.Author.Username)

			// Checking for valid wager
			if player.Chips < wager {
				s.ChannelMessageSend(m.ChannelID,
					fmt.Sprintf("You do not have enough chips to make that wager! Your chip balance is %d.", player.Chips))
				// We return early here because we don't want to start the game if the wager is not valid
				return
			} else if wager <= 0 {
				s.ChannelMessageSend(m.ChannelID, "Your wager must be 1 or higher.")
				return
			}

			GameOn = true

			BlackjackGame = NewBlackjack(player, wager)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Starting a new game of blackjack with %s, wagering %d chips!", m.Author.Username, wager))
			s.ChannelMessageSend(m.ChannelID, BlackjackGame.GetDealerHand())
			BlackjackGame.RunPlayerTurn(s, m)

		} else if strings.ToLower(m.Content) == "!blackjack" {
			// If they entered !blackjack but no wager
			s.ChannelMessageSend(m.ChannelID, "You must place a wager of at least one chip to play!"+
				" Enter \"!blackjack <amount>\" to wager.\nFor example, \"!blackjack 1\" starts a new game wagering 1 "+
				"chip.\nEnter \"!balance\" to see how many chips you have.")
		}

	}

}

// GameOver updates the BlackjackGame's Player to reflect the results of the game, and updates the entry in the database.
// Outputs the game results to Discord, and sets GameOn to False.
func GameOver(s *discordgo.Session, m *discordgo.MessageCreate) {
	message := "Game Over!"

	// If it was a draw
	if BlackjackGame.Wager == 0 {
		message += " Your wager was returned."
	} else {

		if BlackjackGame.Wager > 0 {

			// If the wager is positive they are gaining chips
			message += fmt.Sprintf(" You've gained %d chips!", BlackjackGame.Wager)

		} else {
			// Negative wager, so they lost
			// If the wager brings them to zero, we take pity and keep them at one chip.
			if BlackjackGame.Player.Chips+BlackjackGame.Wager <= 0 {
				message += " Uh oh, looks like you lost the last of your chips! I'll put your total back up to 1, so you can keep playing."
				// They will not be able to wager more chips than they have, so if the wager takes them to zero, we can just add MinChips to the wager to keep them at that number of chips.
				BlackjackGame.Wager += MinChips
			} else {
				// wager *-1, so we get the positive number of chips lost
				message += fmt.Sprintf(" You lost %d chips.", BlackjackGame.Wager*-1)
			}
		}

		// Updating the chip balance for the player
		BlackjackGame.Player.Chips += BlackjackGame.Wager

		message += fmt.Sprintf("\nYour new chip total is: %d", BlackjackGame.Player.Chips)
	}

	dba.UpdatePlayer(BlackjackGame.Player)
	s.ChannelMessageSend(m.ChannelID, message)
	GameOn = false

}

func DisplayLeaderboard(player Player, leaderboardType string, s *discordgo.Session, m *discordgo.MessageCreate) {

	message := strings.ToUpper(leaderboardType) + " LEADERBOARD\n" +
		"================================================\n" +
		"Player Name                               " + strings.ToTitle(leaderboardType) + "\n"

	var leaderboard []Player
	switch leaderboardType {
	case "wins":
		leaderboard = dba.GetLeaderboard(Wins)
	case "chips":
		leaderboard = dba.GetLeaderboard(Chips)
	}

	playerFound := false

	for i, row := range leaderboard {

		if row == player {
			playerFound = true
		}

		var temp int

		switch leaderboardType {
		case "wins":
			temp = row.Wins
		case "chips":
			temp = row.Chips
		}

		if i < 10 {
			message += fmt.Sprintf("%d. %-40s%d\n", i+1, row.Username, temp)
			if i == 9 && !playerFound {
				message += "................................................\n" +
					"................................................\n" +
					"................................................\n"
			}
		} else if row == player {
			message += fmt.Sprintf("%d. %s%d\n", i+1, row.Username, temp)
			message += row.Username
			break
		}

	}

	s.ChannelMessageSend(m.ChannelID, message)

}
