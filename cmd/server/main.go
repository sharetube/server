package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"strings"
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
		envKey:       "SERVER_PORT",
		flagKey:      "server-port",
		defaultValue: 8080,
	}
	host = configVar[string]{
		envKey:       "SERVER_HOST",
		flagKey:      "server-host",
		defaultValue: "0.0.0.0",
	}
	logLevel = configVar[string]{
		envKey:       "SERVER_LOG_LEVEL",
		flagKey:      "server-log-level",
		defaultValue: "INFO",
	}
	membersLimit = configVar[int]{
		envKey:       "SERVER_MEMBERS_LIMIT",
		flagKey:      "server-members-limit",
		defaultValue: 9,
	}
	playlistLimit = configVar[int]{
		envKey:       "SERVER_PLAYLIST_LIMIT",
		flagKey:      "server-playlist-limit",
		defaultValue: 25,
	}
	updatesInterval = configVar[time.Duration]{
		envKey:       "SERVER_UPDATES_INTERVAL",
		flagKey:      "server-updates-interval",
		defaultValue: 5 * time.Second,
	}
	redisPort = configVar[int]{
		envKey:       "REDIS_PORT",
		flagKey:      "redis-port",
		defaultValue: 6379,
	}
	redisHost = configVar[string]{
		envKey:       "REDIS_HOST",
		flagKey:      "redis-host",
		defaultValue: "localhost",
	}
	redisPassword = configVar[string]{
		envKey:       "REDIS_PASSWORD",
		flagKey:      "redis-password",
		defaultValue: "",
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
	pflag.Int(redisPort.flagKey, redisPort.defaultValue, "Redis port")
	pflag.String(redisHost.flagKey, redisHost.defaultValue, "Redis host")
	pflag.String(redisPassword.flagKey, redisPassword.defaultValue, "Redis password")
	pflag.Parse()

	// 2. Bind flags to viper
	viper.BindPFlags(pflag.CommandLine)

	// 3. Set up environment variables prefix and binding
	viper.AutomaticEnv()

	// 4. Set defaults (lowest priority)
	viper.SetDefault(port.envKey, port.defaultValue)
	viper.SetDefault(host.envKey, host.defaultValue)
	viper.SetDefault(logLevel.envKey, logLevel.defaultValue)
	viper.SetDefault(membersLimit.envKey, membersLimit.defaultValue)
	viper.SetDefault(playlistLimit.envKey, playlistLimit.defaultValue)
	viper.SetDefault(updatesInterval.envKey, updatesInterval.defaultValue)
	viper.SetDefault(redisPort.envKey, redisPort.defaultValue)
	viper.SetDefault(redisHost.envKey, redisHost.defaultValue)
	viper.SetDefault(redisPassword.envKey, redisPassword.defaultValue)

	// 5. Create config struct
	config := &app.AppConfig{
		Host:            viper.GetString(host.envKey),
		Port:            viper.GetInt(port.envKey),
		LogLevel:        viper.GetString(logLevel.envKey),
		MembersLimit:    viper.GetInt(membersLimit.envKey),
		PlaylistLimit:   viper.GetInt(playlistLimit.envKey),
		UpdatesInterval: viper.GetDuration(updatesInterval.envKey),
		RedisPort:       viper.GetInt(redisPort.envKey),
		RedisHost:       viper.GetString(redisHost.envKey),
		RedisPassword:   viper.GetString(redisPassword.envKey),
	}

	return config
}

func main() {
	ctx := context.Background()

	appConfig := loadAppConfig()

	logLevel := slog.LevelDebug
	if err := logLevel.UnmarshalText([]byte(strings.ToUpper(appConfig.LogLevel))); err != nil {
		log.Fatal(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	jsonConfig, _ := json.MarshalIndent(appConfig, "", "  ")
	slog.InfoContext(ctx, "starting app with config", "config", string(jsonConfig))

	log.Fatal(app.Run(ctx, appConfig))
}
