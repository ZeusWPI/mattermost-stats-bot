package main

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/rs/zerolog"
)

func main() {

	app := &application{
		logger: zerolog.New(
			zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC822,
			},
		).With().Timestamp().Logger(),
		//messagesCount: prometheus.NewCounter(),
	}

	app.config = loadConfig()
	app.logger.Info().Str("config", fmt.Sprint(app.config)).Msg("")

	setupGracefulShutdown(app)

	// Create a new mattermost client.
	app.mattermostClient = model.NewAPIv4Client(app.config.mattermostServer.String())

	// Login.
	app.mattermostClient.SetToken(app.config.mattermostToken)

	ctx := context.Background()

	if user, resp, err := app.mattermostClient.GetUser(ctx, "me", ""); err != nil {
		app.logger.Fatal().Err(err).Msg("Could not log in")
	} else {
		app.logger.Debug().Interface("user", user).Interface("resp", resp).Msg("")
		app.logger.Info().Msg("Logged in to mattermost")
		app.mattermostUser = user
	}

	// Find and save the bot's team to app struct.
	if team, resp, err := app.mattermostClient.GetTeamByName(ctx, app.config.mattermostTeamName, ""); err != nil {
		app.logger.Fatal().Err(err).Msg("Could not find team. Is this bot a member ?")
	} else {
		app.logger.Debug().Interface("team", team).Interface("resp", resp).Msg("")
		app.mattermostTeam = team
	}

	page := 0
	joined_count := 0
	for { // perPage my ass
		if channels, _, err := app.mattermostClient.GetPublicChannelsForTeam(ctx, app.mattermostTeam.Id, page, 30, ""); err != nil {
			app.logger.Fatal().Err(err).Msg("Could not list all channels, help ?")
		} else {
			if len(channels) == 0 {
				break
			} else {
				for _, channel := range channels {
					if _, _, err := app.mattermostClient.AddChannelMember(ctx, channel.Id, app.mattermostUser.Id); err != nil {
						app.logger.Warn().Err(err).Interface("channel", channel).Msg("Failed to join channel, help ?")
					} else {
						joined_count++
					}
				}
			}
		}

		page++
	}

	app.logger.Info().Interface("count", joined_count).Msg("Joined Channels")

	// Find and save the talking channel to app struct.
	if channel, resp, err := app.mattermostClient.GetChannelByName(
		ctx, app.config.mattermostAdminChannel, app.mattermostTeam.Id, "",
	); err != nil {
		app.logger.Fatal().Err(err).Msg("Could not find channel. Is this bot added to that channel ?")
	} else {
		app.logger.Debug().Interface("channel", channel).Interface("resp", resp).Msg("")
		app.mattermostAdminChannel = channel
	}

	// Send a message (new post).
	sendMsgToTalkingChannel(app, "Hi! StatBot represent.", "")

	// Run the prometheus exporter
	go serveMetrics(app)

	// Listen to live events coming in via websocket.
	listenToEvents(app)
}

func serveMetrics(app *application) {
	collector := newStatsCollector(app)
	prometheus.MustRegister(collector)

	http.Handle("/metrics", promhttp.Handler())
	app.logger.Fatal().AnErr("", http.ListenAndServe(":8000", nil))
}

func setupGracefulShutdown(app *application) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if app.mattermostWebSocketClient != nil {
				app.logger.Info().Msg("Closing websocket connection")
				app.mattermostWebSocketClient.Close()
			}
			app.logger.Info().Msg("Shutting down")
			os.Exit(0)
		}
	}()
}

func sendMsgToTalkingChannel(app *application, msg string, replyToId string) {
	// Note that replyToId should be empty for a new post.
	// All replies in a thread should reply to root.

	post := &model.Post{}
	post.ChannelId = app.mattermostAdminChannel.Id
	post.Message = msg

	post.RootId = replyToId

	ctx := context.Background()

	if _, _, err := app.mattermostClient.CreatePost(ctx, post); err != nil {
		app.logger.Error().Err(err).Str("RootID", replyToId).Msg("Failed to create post")
	}
}

func listenToEvents(app *application) {
	var err error
	failCount := 0
	for {
		app.mattermostWebSocketClient, err = model.NewWebSocketClient4(
			fmt.Sprintf("wss://%s", app.config.mattermostServer.Host+app.config.mattermostServer.Path),
			app.mattermostClient.AuthToken,
		)
		if err != nil {
			app.logger.Warn().Err(err).Msg("Mattermost websocket disconnected, retrying")
			failCount += 1
			// TODO: backoff based on failCount and sleep for a while.
			continue
		}
		app.logger.Info().Msg("Mattermost websocket connected")

		app.mattermostWebSocketClient.Listen()

		for event := range app.mattermostWebSocketClient.EventChannel {
			// Launch new goroutine for handling the actual event.
			// If required, you can limit the number of events beng processed at a time.
			go handleWebSocketEvent(app, event)
		}
	}
}

func handleWebSocketEvent(app *application, event *model.WebSocketEvent) {

	// Ignore other types of events.
	if event.EventType() != model.WebsocketEventPosted {
		return
	}

	app.messagesCount += 1
}
