package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/rodaine/table"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
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
	sess, err := discordgo.New("Bot " + config.Token)
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

	blackjackRegex := regexp.MustCompile(`!blackjack -?\d+`)

	if strings.ToLower(m.Content) == "!balance" {
		chipTotal := dba.GetChipTotal(m.Author.Username)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, your chip total is: %d", m.Author.Username, chipTotal))

	} else if strings.ToLower(m.Content) == "!leaderboard wins" {
		player := dba.FindPlayer(m.Author.Username)
		DisplayLeaderboard(player, "wins", s, m.ChannelID)

	} else if strings.ToLower(m.Content) == "!leaderboard chips" {
		player := dba.FindPlayer(m.Author.Username)
		DisplayLeaderboard(player, "chips", s, m.ChannelID)

	} else if strings.ToLower(m.Content) == "!leaderboard" {
		s.ChannelMessageSend(m.ChannelID, "You can do !leaderboard wins or !leaderboard chips for a leaderboard sorted for each statistic!")

	} else if GameOn {

		// Checking that the message was from the player, it was sent in the correct channel, and it is their turn
		if m.Author.Username == BlackjackGame.Player.Username && m.ChannelID == BlackjackGame.ChannelID && BlackjackGame.IsPlayersTurn {

			if strings.ToLower(m.Content) == "!hit" {

				message := fmt.Sprintln(BlackjackGame.PlayerHit())
				message += BlackjackGame.RunPlayerTurn()
				s.ChannelMessageSend(BlackjackGame.ChannelID, message)

				// If the player's turn is over, which happens when they bust or get 21
				if !BlackjackGame.IsPlayersTurn {
					// If their turn is over because they got 21, run the dealer's turn
					// Dealer doesn't need to go if the player busted
					if BlackjackGame.PlayerHand.Value() == 21 {
						BlackjackGame.RunDealerTurn()
					}
					GameOver(s)
				}

			} else if strings.ToLower(m.Content) == "!stand" {

				s.ChannelMessageSend(m.ChannelID, BlackjackGame.PlayerStand())
				BlackjackGame.RunDealerTurn()
				GameOver(s)

			}
		}

	} else if blackjackRegex.MatchString(strings.ToLower(m.Content)) {
		// If a user enters !blackjack and a bet amount, we begin the game

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

		// Getting the type of the channel the message was sent on
		currentChannel, err := s.Channel(m.ChannelID)
		if err != nil {
			log.Fatal(err)
		}

		// Checking if this message was sent in a guild text channel. If it was, we want to make a thread
		if currentChannel.Type == discordgo.ChannelTypeGuildText {
			// Creating the thread for the game
			currentChannel, err = s.ThreadStart(m.ChannelID, "Blackjack with "+m.Author.Username, discordgo.ChannelTypeGuildPublicThread, 60)
			if err != nil {
				log.Fatal(err)
			}
		}

		// Creating the BlackjackGame and setting the channel ID for the channel it will be played on
		BlackjackGame = NewBlackjack(player, wager)
		BlackjackGame.ChannelID = currentChannel.ID
		GameOn = true

		message := fmt.Sprintf("Starting a new game of blackjack with %s, wagering %d chips!\n", m.Author.Username, wager)
		message += fmt.Sprintln(BlackjackGame.GetDealerHand())
		message += "\n"
		message += BlackjackGame.RunPlayerTurn()

		s.ChannelMessageSend(BlackjackGame.ChannelID, message)

	} else if strings.ToLower(m.Content) == "!blackjack" {
		// If they entered !blackjack but no wager
		s.ChannelMessageSend(m.ChannelID, "You must place a wager of at least one chip to play!"+
			" Enter \"!blackjack <amount>\" to wager.\nFor example, \"!blackjack 1\" starts a new game wagering 1 "+
			"chip.\nEnter \"!balance\" to see how many chips you have.")
	}

}

// GameOver updates the BlackjackGame's Player to reflect the results of the game, and updates the entry in the database.
// Outputs the game results to Discord, and sets GameOn to False.
func GameOver(s *discordgo.Session) {

	message := BlackjackGame.Results()

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
				message += "\n\nUh oh, looks like you lost the last of your chips! I'll put your total back up to 1, so you can keep playing."
				// They will not be able to wager more chips than they have, so if the wager takes them to zero, we can just add MinChips to the wager to keep them at that number of chips.
				BlackjackGame.Wager += MinChips
			} else {
				// wager *-1, so we get the positive number of chips lost
				message += fmt.Sprintf(" You lost %d chips.", BlackjackGame.Wager*-1)
			}
		}

		// Updating the chip balance for the player
		BlackjackGame.Player.Chips += BlackjackGame.Wager

		message += fmt.Sprintf("\n\nYour new chip total is: %d", BlackjackGame.Player.Chips)
	}

	dba.UpdatePlayer(BlackjackGame.Player)
	s.ChannelMessageSend(BlackjackGame.ChannelID, message)
	GameOn = false

}

func DisplayLeaderboard(player Player, leaderboardType string, s *discordgo.Session, channelID string) {

	var tbl table.Table
	var sb strings.Builder

	// Writing the table name
	sb.WriteString(strings.ToUpper(leaderboardType) + " LEADERBOARD\n")
	sb.WriteString("================================================\n")

	var leaderboard []Player

	// Getting the proper headers and leaderboard
	switch leaderboardType {
	case "wins":
		leaderboard = dba.GetLeaderboard(Wins)
		tbl = table.New("RANK", "PLAYER", "WINS", "TIES", "LOSSES", "CHIPS")
	case "chips":
		leaderboard = dba.GetLeaderboard(Chips)
		tbl = table.New("RANK", "PLAYER", "CHIPS", "WINS", "TIES", "LOSSES")
	}

	// Looping through the rows, displaying top 5 and player who requested leaderboard
	for i, row := range leaderboard {

		if i < 5 || row == player {

			switch leaderboardType {
			case "wins":
				tbl.AddRow(i+1, row.Username, row.Wins, row.Ties, row.Losses, row.Chips)
			case "chips":
				tbl.AddRow(i+1, row.Username, row.Chips, row.Wins, row.Ties, row.Losses)
			}

		}
	}

	// Setting the table to write to the string builder
	tbl.WithWriter(&sb)
	tbl.Print()

	// Printing this inside ``` ``` wrapping to make it block text in discord, so formatting is not messed up
	s.ChannelMessageSend(channelID, "```"+sb.String()+"```")

}
