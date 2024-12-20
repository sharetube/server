package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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
	secret = configVar[string]{
		envKey:       "SERVER_SECRET",
		flagKey:      "secret",
		defaultValue: "",
	}
	logPath = configVar[string]{
		envKey:       "SERVER_LOG_PATH",
		flagKey:      "log-path",
		defaultValue: "/var/log/sharetube/server.log",
	}
	port = configVar[int]{
		envKey:       "SERVER_PORT",
		flagKey:      "port",
		defaultValue: 80,
	}
	host = configVar[string]{
		envKey:       "SERVER_HOST",
		flagKey:      "host",
		defaultValue: "0.0.0.0",
	}
	logLevel = configVar[string]{
		envKey:       "SERVER_LOG_LEVEL",
		flagKey:      "log-level",
		defaultValue: "INFO",
	}
	membersLimit = configVar[int]{
		envKey:       "SERVER_MEMBERS_LIMIT",
		flagKey:      "members-limit",
		defaultValue: 9,
	}
	playlistLimit = configVar[int]{
		envKey:       "SERVER_PLAYLIST_LIMIT",
		flagKey:      "playlist-limit",
		defaultValue: 25,
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
	pflag.String(secret.flagKey, secret.defaultValue, "Server secret")
	pflag.Int(port.flagKey, port.defaultValue, "Server port")
	pflag.String(host.flagKey, host.defaultValue, "Server host")
	pflag.String(logLevel.flagKey, logLevel.defaultValue, "Logging level")
	pflag.String(logPath.flagKey, logPath.defaultValue, "Log file path")
	pflag.Int(membersLimit.flagKey, membersLimit.defaultValue, "Maximum number of members in the room")
	pflag.Int(playlistLimit.flagKey, playlistLimit.defaultValue, "Maximum number of videos in the playlist")
	pflag.Int(redisPort.flagKey, redisPort.defaultValue, "Redis port")
	pflag.String(redisHost.flagKey, redisHost.defaultValue, "Redis host")
	pflag.String(redisPassword.flagKey, redisPassword.defaultValue, "Redis password")
	pflag.Parse()

	// 2. Bind flags to viper
	viper.BindPFlags(pflag.CommandLine)

	// 3. Set up environment variables prefix and binding
	viper.AutomaticEnv()

	// 4. Set defaults (lowest priority)
	viper.SetDefault(secret.envKey, secret.defaultValue)
	viper.SetDefault(port.envKey, port.defaultValue)
	viper.SetDefault(host.envKey, host.defaultValue)
	viper.SetDefault(logLevel.envKey, logLevel.defaultValue)
	viper.SetDefault(logPath.envKey, logPath.defaultValue)
	viper.SetDefault(membersLimit.envKey, membersLimit.defaultValue)
	viper.SetDefault(playlistLimit.envKey, playlistLimit.defaultValue)
	viper.SetDefault(redisPort.envKey, redisPort.defaultValue)
	viper.SetDefault(redisHost.envKey, redisHost.defaultValue)
	viper.SetDefault(redisPassword.envKey, redisPassword.defaultValue)

	// 5. Create config struct
	config := &app.AppConfig{
		Secret:        viper.GetString(secret.envKey),
		Host:          viper.GetString(host.envKey),
		Port:          viper.GetInt(port.envKey),
		LogLevel:      viper.GetString(logLevel.envKey),
		LogPath:       viper.GetString(logPath.envKey),
		MembersLimit:  viper.GetInt(membersLimit.envKey),
		PlaylistLimit: viper.GetInt(playlistLimit.envKey),
		RedisPort:     viper.GetInt(redisPort.envKey),
		RedisHost:     viper.GetString(redisHost.envKey),
		RedisPassword: viper.GetString(redisPassword.envKey),
	}

	return config
}

func main() {
	ctx := context.Background()

	appConfig := loadAppConfig()

	jsonConfig, _ := json.MarshalIndent(appConfig, "", "  ")
	// slog.InfoContext(ctx, "starting app with config", "config", string(jsonConfig))
	fmt.Printf("starting app with config: %s\n", jsonConfig)

	log.Fatal(app.Run(ctx, appConfig))
}
