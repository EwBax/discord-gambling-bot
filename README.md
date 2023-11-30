# Discordgo Gambling Bot

## Description
GamblingBot is a Discord bot written with Golang using the discordgo library from github.com/bwmarrin/discordgo. 
The bot enables members of any server it is a member of to play blackjack against the bot. The bot uses a SQLite database
to track stats for each player.

## Setup
The bot uses discord slash-commands to operate. The bot will register these commands to any server it is a member of on
startup. 

To build and activate the bot, you must complete a few steps.
1. Make sure you have Golang installed.
2. Follow the steps required to create a bot application on Discord, and invite the bot to your server.
3. Create a copy of "config_template.json" in the same directory, and store your bot token and path to the SQLite database.
   (unless you've moved it, the database is in the main project directory)
4. You must have Cmake installed to build the program successfully, due to the use of the library github.com/mattn/go-sqlite3.
5. Open a cmd/terminal window in the main project directory.
6. Run the following commands:\
`$env:CGO_ENABLED=1`\
`go build .`
7. Double click GamblingBot.exe to start the bot!

