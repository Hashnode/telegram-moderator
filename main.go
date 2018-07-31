package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"gopkg.in/telegram-bot-api.v4"
)

func main() {
	var err error
	var server *http.Server
	var stopServer = false
	// Declare flags
	botToken := flag.String("token", "", "telegram bot token")
	debug := flag.Bool("debug", true, "turn debug bode on/off")
	webhookBaseURL := flag.String("webhookBaseURL", "", "Base URL for webhook")
	port := flag.String("port", "80", "port to listen")
	charlength := flag.Int("charlength", 20, "max length for username/password")
	illegalChars := flag.String("illegalchars", "", "Illegal chars in name field")
	flag.Parse()

	if *botToken == "" {
		fmt.Printf("usage:\n hashnodebot -token <token>\n")
		os.Exit(1)
	}

	bot, err := tgbotapi.NewBotAPI(*botToken)
	if err != nil {
		log.Fatalf("Initializing bot with the given token failed : %v\n", err)
	}
	bot.Debug = *debug
	log.Printf("Intialized with username %s\n", bot.Self.UserName)

	var updates tgbotapi.UpdatesChannel

	// If a webhook url is given start the bot in webhook mode
	if *webhookBaseURL != "" {
		log.Println("Starting Listener On..", *webhookBaseURL)
		err := startWithWebHook(bot, *webhookBaseURL)
		if err != nil {
			log.Fatal(err)
		}
		updates = bot.ListenForWebhook("/" + bot.Token)
		server, err = startServer(*port)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		config := startWithPolling(bot, 1)
		updates, err = bot.GetUpdatesChan(*config)
		if err != nil {
			log.Fatal(err)
		}

	}

	// Exit gracefully on interupt
	go exitGracefully(func(done chan bool) {
		if server != nil {
			server.Shutdown(context.Background())
		}
		done <- true
		return
	})

	for update := range updates {
		if update.Message == nil {
			continue
		}
		if stopServer {
			break
		}

		// Check for spammers
		go func(update tgbotapi.Update) {
			if update.Message.NewChatMembers != nil {

				// Remove this service message
				// Then check if the newly joined member is a spammer
				bot.DeleteMessage(tgbotapi.DeleteMessageConfig{
					ChatID:    update.Message.Chat.ID,
					MessageID: update.Message.MessageID,
				})

				for _, user := range *update.Message.NewChatMembers {
					if checkIfSpammer(user.UserName, user.FirstName+user.LastName, *charlength, *illegalChars) {
						bot.KickChatMember(tgbotapi.KickChatMemberConfig{
							ChatMemberConfig: tgbotapi.ChatMemberConfig{
								UserID: user.ID,
								ChatID: update.Message.Chat.ID,
							},
						})
					}
				}
			} else if update.Message.LeftChatMember != nil {
				// Just delete the service message
				bot.DeleteMessage(tgbotapi.DeleteMessageConfig{
					ChatID:    update.Message.Chat.ID,
					MessageID: update.Message.MessageID,
				})
			}
		}(update)

	}
}

func startWithWebHook(bot *tgbotapi.BotAPI, webhookURL string) error {
	_, err := bot.SetWebhook(tgbotapi.NewWebhook(webhookURL + "/" + bot.Token))
	if err != nil {
		return err
	}
	info, err := bot.GetWebhookInfo()
	if err != nil {
		return err
	}

	if info.LastErrorDate != 0 {
		return fmt.Errorf("Callback to telegram failed: %s ", info.LastErrorMessage)
	}
	return nil
}

func startWithPolling(bot *tgbotapi.BotAPI, timeout int) *tgbotapi.UpdateConfig {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = timeout
	return &u
}

func startServer(port string) (*http.Server, error) {
	url := "0.0.0.0:" + port
	listener, err := net.Listen("tcp", url)
	if err != nil {
		return nil, err
	}

	server := &http.Server{
		ReadTimeout:    time.Second * 30,
		WriteTimeout:   time.Second * 60,
		MaxHeaderBytes: 1 << 20,
	}

	go server.Serve(listener)
	return server, nil

}

func checkIfSpammer(username, name string, length int, illegalChars string) bool {
	if illegalChars != "" && strings.Contains(name, illegalChars) {
		return true
	}
	if len(username) > length || len(name) > length {
		return true
	}
	return false
}

func exitGracefully(handleShutdown func(chan bool)) {
	signalChan := make(chan os.Signal, 1)
	cleanUpDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		sig := <-signalChan
		fmt.Printf("\nRecived %v\n", sig)
		// handle cleanup tasks
		handleShutdown(cleanUpDone)

		fmt.Printf("Cleanup Completed...Now shutting down\n")
		os.Exit(0)
	}()
	<-cleanUpDone
}
