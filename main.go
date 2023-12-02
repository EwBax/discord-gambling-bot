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
	dba    DBA
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

	// BlackjackGamesMap Global variable slice of ongoing games of blackjack
	BlackjackGamesMap = make(map[string]*Blackjack)

	// commands is a list of the application commands this bot uses
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
					Name:        "wager",
					Description: "The amount of chips you want to wager.",
					Required:    true,
					MinValue:    &minWager,
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "force",
					Description: "If you are in another game, forfeits that wager and forces it to stop before starting a new one.",
					Required:    false,
				},
			},
		},
	}

	// commandHandlers is a list of the command handlers for each command
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"balance": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			chipTotal := dba.GetChipTotal(i.Member.User.Username)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: GetLeaderboard(i.Member.User.Username, option),
				},
			})

		},
		"blackjack": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			// Getting options and storing in map
			options := i.ApplicationCommandData().Options
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			player := dba.FindPlayer(i.Member.User.Username)

			// Checking if there is a game being played in this channel
			game := FindGameByChannelID(i.ChannelID)
			// There is a game being played on this channel
			if game != nil {

				// If the player of the ongoing game is the player sending the command
				if game.Player == player {
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "We're already playing a game here!",
						},
					})
				} else {
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf(
								"I'm currently playing a game with %s in this channel. Please try another channel.",
								game.Player.Username,
							),
						},
					})
				}
				return

			}

			// Getting the wager amount from the command option. It is validated to be an integer > 1 by the command settings
			wager := int(optionMap["wager"].IntValue())

			// Checking the player doesn't have enough chips
			if player.Chips < wager {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf(
							"You don't have enough chips for that wager! Your current balance is: %d",
							player.Chips,
						),
					},
				})
				return
			}

			// Checking if the player starting the game is currently in a game already
			game, ok := BlackjackGamesMap[player.Username]
			if ok {
				var force *discordgo.ApplicationCommandInteractionDataOption
				force, ok = optionMap["force"]
				// If the player is already in a game
				if ok && force.BoolValue() {

					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf(
								"Stopping your other game to start a new one! Your previous wager of %d was forfeited.",
								game.Wager,
							),
						},
					})
					startingChips := player.Chips
					// subtracting the wager unless it would put them below 1 chip
					if player.Chips-game.Wager >= 1 {
						player.Chips -= game.Wager
					} else {
						player.Chips = 1
					}
					// only updating the player if their chips actually changed
					if startingChips != player.Chips {
						dba.UpdatePlayer(player)
					}

				} else {

					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "You're currently in a game elsewhere! Finish that game first, or use the force flag to start a new game.",
						},
					})
					return

				}
			} else {
				// The player is not currently in a game
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf(
							"Starting a new game of blackjack with %s!",
							player.Username,
						),
					},
				})
			}

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
			newGame := NewBlackjack(player, wager)
			newGame.ChannelID = gameChannel.ID
			// Adding the game to the map
			BlackjackGamesMap[player.Username] = &newGame

			// Creating the message to display to the player at the start of the game
			message := newGame.GetDealerHand() + "\n\n"
			message += newGame.RunPlayerTurn()

			// If the player gets dealt a 21
			if !newGame.IsPlayersTurn {
				_, _ = s.ChannelMessageSend(newGame.ChannelID, message)
				newGame.RunDealerTurn()
				GameOver(newGame)
			} else {
				DisplayHitStandButtons(newGame.ChannelID, message)
			}

		},
		"hit": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			// find the game that is being played in this channel
			game := FindGameByChannelID(i.ChannelID)

			// if the game is nil we need to remove the buttons because the game is over now
			if game == nil {
				RemoveComponentsFromMessage(i.ChannelID, i.Message.ID, i.Message.Content)
				return
			}

			// Checking that the message was from the correct player
			username := i.Member.User.Username

			// If the game it was from is over now, or it was from another user, discard the interaction
			if username != game.Player.Username {
				AcknowledgeInteraction(i)
				return
			}

			// Removing the buttons from the message so they can't be used again.
			RemoveComponentsFromMessage(i.ChannelID, i.Message.ID, i.Message.Content)

			// If it was from the player
			game.Hit(&game.PlayerHand)
			message := "You chose to hit!\n"
			message += game.RunPlayerTurn()

			// If the player's turn is now over, which happens if they bust or get 21
			if !game.IsPlayersTurn {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: message,
					},
				})
				// Dealer doesn't need to go if the player busted
				if game.PlayerHand.Value() == 21 {
					game.IsPlayersTurn = false
					game.RunDealerTurn()
				}
				GameOver(*game)
			} else {
				// The player's turn is not over
				RespondHitStandButtons(i, message)
			}

		},
		"stand": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			game := FindGameByChannelID(i.ChannelID)

			// if the game is nil we need to remove the buttons because the game is over now
			if game == nil {
				RemoveComponentsFromMessage(i.ChannelID, i.Message.ID, i.Message.Content)
				return
			}

			//Checking that the message was from the correct player
			username := i.Member.User.Username

			// If it was from another user, discard the interaction
			if username != game.Player.Username {
				AcknowledgeInteraction(i)
				return
			}

			// Removing the buttons from the message so they can't be used again.
			RemoveComponentsFromMessage(i.ChannelID, i.Message.ID, i.Message.Content)

			// If it was from the player
			game.IsPlayersTurn = false
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "\nYou stand! It is now the dealer's turn.",
				},
			})
			game.RunDealerTurn()
			GameOver(*game)

		},
	}
)

// AcknowledgeInteraction sends an empty response to an interaction then deletes it. To acknowledge the interaction without
// leaving a response
func AcknowledgeInteraction(i *discordgo.InteractionCreate) {
	// Empty response
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "",
		},
	})
	// Deleting the response
	_ = s.InteractionResponseDelete(i.Interaction)
}

// RemoveComponentsFromMessage edits a message to remove any components from it, leaving just the content.
func RemoveComponentsFromMessage(channelID string, messageID string, content string) {
	_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Content:    &content,
		ID:         messageID,
		Channel:    channelID,
		Components: []discordgo.MessageComponent{},
	})

	if err != nil {
		log.Fatal(err)
	}
}

// FindGameByChannelID is a helper function to find a game in the BlackjackGamesMap by channelID
func FindGameByChannelID(channelID string) *Blackjack {

	// find the game that is being played in this channel
	var game *Blackjack
	for _, game = range BlackjackGamesMap {
		if game.ChannelID == channelID {
			// return when we find the right game
			return game
		}
	}

	// Return nil because we did not find the game
	return nil
}

// DisplayHitStandButtons takes a channelID and a message and sends the message with hit and stand buttons added.
func DisplayHitStandButtons(channelID string, message string) {

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: message,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Hit",
						Style:    discordgo.SuccessButton,
						CustomID: "hit",
					},
					discordgo.Button{
						Label:    "Stand",
						Style:    discordgo.DangerButton,
						CustomID: "stand",
					},
				},
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}

// RespondHitStandButtons takes an interaction and responds to it with hit and stand buttons
func RespondHitStandButtons(i *discordgo.InteractionCreate, message string) {

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Hit",
							Style:    discordgo.SuccessButton,
							CustomID: "hit",
						},
						discordgo.Button{
							Label:    "Stand",
							Style:    discordgo.DangerButton,
							CustomID: "stand",
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func init() {

	// Adding a handler to the session to handle InteractionCreate events (slash command)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// Checking the interaction type
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			// Calling the command handler for the command that was sent, passing the session and the interaction
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionMessageComponent:
			if h, ok := commandHandlers[i.MessageComponentData().CustomID]; ok {
				h(s, i)
			}
		}
	})

}

func main() {

	err := s.Open()

	if err != nil {
		log.Fatal(err)
	}

	// Adding the commands
	for _, cmd := range commands {
		// Using empty guildID to create command globally
		_, err = s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", cmd.Name, err)
		}
	}

	// Sets the intent for the session
	s.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	// deferring s.Close() until this function returns. Wrapped in function to handle error
	defer func(s *discordgo.Session) {
		err := s.Close()
		if err != nil {

		}
	}(s)

	fmt.Println("GamblingBot is online.")

	// This makes the bot continue to run a termination signal is sent
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

}

// GameOver updates the BlackjackGame's Player to reflect the results of the game, and updates the entry in the database.
// Outputs the game results to Discord, and sets GameOn to False.
func GameOver(game Blackjack) {

	message := game.Results()

	// If it was a draw
	if game.Wager == 0 {
		message += " Your wager was returned."
	} else {

		if game.Wager > 0 {

			// If the wager is positive they are gaining chips
			message += fmt.Sprintf(" You've gained %d chips!", game.Wager)

		} else {
			// Negative wager, so they lost
			// If the wager brings them to zero, we take pity and keep them at one chip.
			if game.Player.Chips+game.Wager <= 0 {
				message += "\n\nUh oh, looks like you lost the last of your chips! I'll put your total back up to 1, so you can keep playing."
				// They will not be able to wager more chips than they have, so if the wager takes them to zero, we can just add MinChips to the wager to keep them at that number of chips.
				game.Wager += MinChips
			} else {
				// wager *-1, so we get the positive number of chips lost
				message += fmt.Sprintf(" You lost %d chips.", game.Wager*-1)
			}
		}

		// Updating the chip balance for the player
		game.Player.Chips += game.Wager

		message += fmt.Sprintf("\n\nYour new chip total is: %d", game.Player.Chips)
	}
	// Updating the player entry
	dba.UpdatePlayer(game.Player)
	// Removing the game from the map since it is done now
	delete(BlackjackGamesMap, game.Player.Username)
	_, _ = s.ChannelMessageSend(game.ChannelID, message)

}

func GetLeaderboard(username string, leaderboardType string) string {

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
