package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Whether the bot is currently playing a game with someone
var GAME_ON = false

type Card struct {
	Rank string
	Suit string
}

func (c Card) String() string {
	return c.Rank + " of " + c.Suit
}

var ranks = map[string]int{"Ace": 1, "Two": 2, "Three": 3, "Four": 4, "Five": 5, "Six": 6, "Seven": 7, "Eight": 8, "Nine": 9, "Ten": 10, "Jack": 10, "Queen": 10, "King": 10}

type Deck []Card

func NewDeck() Deck {
	var deck Deck

	suits := []string{"Hearts", "Clubs", "Diamonds", "Spades"}

	for _, suit := range suits {
		for rank := range ranks {
			deck = append(deck, Card{rank, suit})
		}
	}

	return deck

}

func (d Deck) String() string {

	message := ""

	for i, c := range d {
		message += c.String()
		if i < len(d)-1 {
			message += ",\n"
		}
	}

	return message

}

func (d Deck) Shuffle() {

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := range d {
		j := rng.Intn(len(d))
		d[i], d[j] = d[j], d[i]
	}

}

func (d *Deck) DealCard() Card {

	if len(*d) <= 0 {
		panic("Deck has no cards.")
	}

	// getting the last card
	card := (*d)[len(*d)-1]

	// slicing off the last card
	*d = (*d)[:len(*d)-1]

	return card

}

type Hand []Card

func (h Hand) Value() int {

	value := 0
	numAces := 0

	// Getting the value and the number of aces in the deck
	for i := range h {
		value += ranks[h[i].Rank]
		if h[i].Rank == "Ace" {
			numAces++
		}
	}

	// If there are more aces and using one as a 10 would not cause the hand to bust, increase value by 10
	for numAces < 0 && value+9 == 21 {
		value += 9
		numAces--
	}

	return value

}

type Blackjack struct {
	PlayerName string
	CardDeck   Deck
	PlayerHand Hand
	DealerHand Hand
}

func NewBlackjack(playerName string) Blackjack {

	newGame := Blackjack{playerName, NewDeck(), make(Hand, 0), make(Hand, 0)}

	newGame.CardDeck.Shuffle()
	newGame.CardDeck.Shuffle()
	newGame.CardDeck.Shuffle()
	newGame.CardDeck.Shuffle()

	// Dealing player hand
	newGame.PlayerHand = append(newGame.PlayerHand, newGame.CardDeck.DealCard())
	fmt.Print(strconv.Itoa(len(newGame.CardDeck)))
	newGame.PlayerHand = append(newGame.PlayerHand, newGame.CardDeck.DealCard())
	fmt.Print(strconv.Itoa(len(newGame.CardDeck)))

	// Dealer hand
	newGame.DealerHand = append(newGame.PlayerHand, newGame.CardDeck.DealCard())
	newGame.DealerHand = append(newGame.PlayerHand, newGame.CardDeck.DealCard())

	return newGame

}

var BlackjackGame Blackjack

func main() {

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

	if GAME_ON {

		if BlackjackGame.PlayerHand.Value() < 21 {
			PromptPlayer(s, m)
		}

	} else {

		// If a user enters !blackjack we begin the game
		if strings.ToLower(m.Content) == "!blackjack" {
			GAME_ON = true
			BlackjackGame = NewBlackjack(m.Author.Username)
			s.ChannelMessageSend(m.ChannelID, "Starting a new game of blackjack with "+m.Author.Username+"!")
		}

	}

}

func PromptPlayer(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := BlackjackGame.PlayerName + " your hand is:\n"

	// Making a message with the player's hand, adding each card to the message
	for i, card := range BlackjackGame.PlayerHand {
		message += "\t" + card.Rank + " of " + card.Suit
		if i < len(BlackjackGame.PlayerHand)-1 {
			message += ","
		}
		message += "\n"
	}

	message += "The value of your hand is: " + strconv.Itoa(BlackjackGame.PlayerHand.Value()) + "\n\n"

	message += "Enter !hit to hit, !hold to hold."

	s.ChannelMessageSend(m.ChannelID, message)
}
