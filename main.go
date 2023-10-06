// From this tutorial: https://www.youtube.com/watch?v=XuFq7NW3ii4

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	sess, err := discordgo.New("Bot MTE1NzAzOTk4NTgxNjUxNDcwMQ.GOVQIL.KyV-XFZB0f1ylsYC0PQqOwFH5yw5vwgwWVupBM")

	if err != nil {
		log.Fatal(err)
	}

	// Handles a message being created in discord
	sess.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Checks that the author was a user other than the bot
		if m.Author.ID == s.State.User.ID {
			return
		}

		// If a user enters Hello the bot replies with World! immediately after
		if m.Content == "Hello" {
			s.ChannelMessageSend(m.ChannelID, "World!")
		}
	})

	// Sets the intent for the session
	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = sess.Open()

	if err != nil {
		log.Fatal(err)
	}

	defer sess.Close()

	fmt.Println("The bot is online.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
