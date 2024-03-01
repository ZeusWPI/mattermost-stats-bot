# Mattermost Bot Sample

## Overview

This Bot joins all public channels and keep count of how many messages have been sent.
This count is exposed in a `/metrics` endpoint that can be queried by prometheus.

This Bot was tested with Mattermost server version 9.5.0

## Setup Server Environment

Add a bot user with member access to all public channels.
Copy `example.env` to `.env` and fill in the bot token (obtained from the previous step), team name, etc.

```sh
. .env && do build -o mm-bot && ./mm-bot
```

## Test the Bot

1 - Log in to your Mattermost server.

3 - Post a message in a channel. The count in the metrics endpoint should increase.

## Stop the Bot

1 - In the terminal window, press `CTRL+C` to stop the bot.
