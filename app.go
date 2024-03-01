package main

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/rs/zerolog"
)

// application struct to hold the dependencies for our bot
type application struct {
	config                    config
	logger                    zerolog.Logger
	mattermostClient          *model.Client4
	mattermostWebSocketClient *model.WebSocketClient
	mattermostUser            *model.User
	mattermostAdminChannel    *model.Channel
	mattermostTeam            *model.Team

	messagesCount uint64
}
