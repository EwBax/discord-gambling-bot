package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/rodaine/table"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// https://github.com/bwmarrin/discordgo/blob/master/examples/slash_commands/main.go#L23

var (
	// s is the session connection
	s *discordgo.Session
	// The database adapter
	dba DBA
	// GameOn Whether the bot is currently playing a game with someone
	GameOn = false
	Config Configuration
)

func init() {
	Config = GetConfig()
	// Opening the database connection
	dba.OpenConnection(Config.DbPath)

	var err error
	// Creating a new Discord session using the bot token
	s, err = discordgo.New("Bot " + Config.Token)
	if err != nil {
		log.Fatal(err)
	}

}

// Adding discord slash commands and handlers
var (
	minWager = 1.0

	// BlackjackGame Global variable for a game of blackjack
	BlackjackGame Blackjack

	commands = []*discordgo.ApplicationCommand{
		{
			// Name of the command
			Name: "balance",
			// Description of the command.
			Description: "See your current balance of chips.",
		},
		{
			Name:        "leaderboard",
			Description: "View the leaderboard, sorted by either wins or chips.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "type",
					Description: "Enter \"wins\" to sort by wins, or \"chips\" to sort by chips",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "wins",
							Value: "wins",
						},
						{
							Name:  "chips",
							Value: "chips",
						},
					},
				},
			},
		},
		{
			Name:        "blackjack",
			Description: "Start a game of blackjack against the bot!",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "bet",
					Description: "The amount of chips you want to wager.",
					Required:    true,
					MinValue:    &minWager,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"balance": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			chipTotal := dba.GetChipTotal(i.Member.User.Username)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				// Ignore type for now, they will be discussed in "responses"
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf(
						"%s, your chip total is: %d",
						i.Member.User.Username,
						chipTotal,
					),
				},
			})
		},
		"leaderboard": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			// Access options in the order provided by the user.
			option := strings.ToLower(i.ApplicationCommandData().Options[0].StringValue())

			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				// Ignore type for now, they will be discussed in "responses"
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: DisplayLeaderboard(i.Member.User.Username, option),
				},
			})

		},
		"blackjack": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			// Checking if we're in a game, we can only play one game at a time in guild, but can play multiple over direct message
			if GameOn {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					// Ignore type for now, they will be discussed in "responses"
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "I'm currently in a game, please try again in a moment!",
					},
				})
				return
			}

			// Getting the wager amount from the command option. It is validated to be an integer > 1 by the command settings
			wager := int(i.ApplicationCommandData().Options[0].IntValue())

			player := dba.FindPlayer(i.Member.User.Username)

			// Getting the type of the channel the message was sent on
			currentChannel, _ := s.Channel(i.ChannelID)
			gameChannel := currentChannel

			// Checking if this message was sent in a guild text channel. If it was, we want to make a thread
			if currentChannel.Type == discordgo.ChannelTypeGuildText {
				// Creating the thread for the game
				var err error
				gameChannel, err = s.ThreadStart(i.ChannelID, "Blackjack with "+i.Member.User.Username, discordgo.ChannelTypeGuildPublicThread, 60)
				if err != nil {
					log.Fatal(err)
				}
			}

			// Creating the game and setting the game channel
			BlackjackGame = NewBlackjack(player, wager)
			BlackjackGame.ChannelID = gameChannel.ID
			GameOn = true

			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				// Ignore type for now, they will be discussed in "responses"
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf(
						"Starting a new game of blackjack with %s!",
						i.Member.User.Username,
					),
				},
			})

			message := BlackjackGame.GetDealerHand() + "\n\n"
			message += BlackjackGame.RunPlayerTurn()

			_, _ = s.ChannelMessageSend(BlackjackGame.ChannelID, message)

		},
	}
)

func init() {
	// Adding a handler to the session to handle InteractionCreate events (slash command)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// Calling the command handler for the command that was sent, passing the session and the interaction
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {

	GameOn = false

	err := s.Open()

	if err != nil {
		log.Fatal(err)
	}

	// Adding the commands
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		var cmd *discordgo.ApplicationCommand
		// Using empty guildID to create command globally
		cmd, err = s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	// Handles a message being created in discord
	s.AddHandler(MessageReceived)

	// Sets the intent for the session
	s.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	// deferring s.Close() until this function returns. Wrapped in function to handle error
	defer func(s *discordgo.Session) {
		err := s.Close()
		if err != nil {

		}
	}(s)

	fmt.Println("The bot is online.")

	// This makes the bot continue to run a termination signal is sent
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

}

func MessageReceived(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Checks that the author was a user other than the bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	if GameOn {

		// Checking that the message was from the player, it was sent in the correct channel, and it is their turn
		if m.Author.Username == BlackjackGame.Player.Username && m.ChannelID == BlackjackGame.ChannelID && BlackjackGame.IsPlayersTurn {

			if strings.ToLower(m.Content) == "!hit" {

				message := fmt.Sprintln(BlackjackGame.PlayerHit())
				message += BlackjackGame.RunPlayerTurn()
				_, _ = s.ChannelMessageSend(BlackjackGame.ChannelID, message)

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

				_, _ = s.ChannelMessageSend(m.ChannelID, BlackjackGame.PlayerStand())
				BlackjackGame.RunDealerTurn()
				GameOver(s)

			}
		}

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
	_, _ = s.ChannelMessageSend(BlackjackGame.ChannelID, message)
	GameOn = false

}

func DisplayLeaderboard(username string, leaderboardType string) string {

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

		if i < 5 || row.Username == username {

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
	return "```" + sb.String() + "```"

}
