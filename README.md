# Telegram Moderator Bot

## What it does?
Kicks out users from Telegram channels whose name/username match a particular pattern

## How to run ?
Make sure you have working golang installation, Install https://github.com/golang/dep
and run a `dep ensure`
then,

` go run main.go <flags> `
```
Usage:
  -charlength int
    	max length for username/name (default 20)
  -debug
    	turn debug bode on/off (default true)
  -port string
    	port to listen (default "80")
  -token string
    	telegram bot token
  -webhookBaseURL string
    	Base URL for webhook
 ```

