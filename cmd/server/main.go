package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/sharetube/server/internal/app"
)

type configVar[T any] struct {
	envKey       string
	flagKey      string
	defaultValue T
}

var (
	port = configVar[int]{
		envKey:       "PORT",
		flagKey:      "port",
		defaultValue: 8080,
	}
	host = configVar[string]{
		envKey:       "HOST",
		flagKey:      "host",
		defaultValue: "0.0.0.0",
	}
	logLevel = configVar[string]{
		envKey:       "LOG_LEVEL",
		flagKey:      "log-level",
		defaultValue: "info",
	}
	membersLimit = configVar[int]{
		envKey:       "MEMBERS_LIMIT",
		flagKey:      "members-limit",
		defaultValue: 9,
	}
	playlistLimit = configVar[int]{
		envKey:       "PLAYLIST_LIMIT",
		flagKey:      "playlist-limit",
		defaultValue: 25,
	}
	updatesInterval = configVar[time.Duration]{
		envKey:       "UPDATES_INTERVAL",
		flagKey:      "updates-interval",
		defaultValue: 5 * time.Second,
	}
)

func loadAppConfig() *app.AppConfig {
	// 1. Define flags
	pflag.Int(port.flagKey, port.defaultValue, "Server port")
	pflag.String(host.flagKey, host.defaultValue, "Server host")
	pflag.String(logLevel.flagKey, logLevel.defaultValue, "Logging level")
	pflag.Int(membersLimit.flagKey, membersLimit.defaultValue, "Maximum number of members in the room")
	pflag.Int(playlistLimit.flagKey, playlistLimit.defaultValue, "Maximum number of videos in the playlist")
	pflag.Duration(updatesInterval.flagKey, updatesInterval.defaultValue, "Interval between updates")
	pflag.Parse()

	// 2. Bind flags to viper
	viper.BindPFlags(pflag.CommandLine)

	// 3. Set up environment variables prefix and binding
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()

	// 4. Set defaults (lowest priority)
	viper.SetDefault(port.envKey, port.defaultValue)
	viper.SetDefault(host.envKey, host.defaultValue)
	viper.SetDefault(logLevel.envKey, logLevel.defaultValue)
	viper.SetDefault(membersLimit.envKey, membersLimit.defaultValue)
	viper.SetDefault(playlistLimit.envKey, playlistLimit.defaultValue)
	viper.SetDefault(updatesInterval.envKey, updatesInterval.defaultValue)

	// 5. Create config struct
	config := &app.AppConfig{
		Host:            viper.GetString(host.envKey),
		Port:            viper.GetInt(port.envKey),
		LogLevel:        viper.GetString(logLevel.envKey),
		MembersLimit:    viper.GetInt(membersLimit.envKey),
		PlaylistLimit:   viper.GetInt(playlistLimit.envKey),
		UpdatesInterval: viper.GetDuration(updatesInterval.envKey),
	}

	return config
}

func main() {
	ctx := context.Background()
	appConfig := loadAppConfig()

	jsonConfig, _ := json.MarshalIndent(appConfig, "", "  ")
	fmt.Println(string(jsonConfig))

	app.Run(ctx, appConfig)
}
