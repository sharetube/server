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
	// todo: move to pkg
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

	viper.BindPFlags(pflag.CommandLine)

	viper.BindEnv(secret.flagKey, secret.envKey)
	viper.BindEnv(port.flagKey, port.envKey)
	viper.BindEnv(host.flagKey, host.envKey)
	viper.BindEnv(logLevel.flagKey, logLevel.envKey)
	viper.BindEnv(logPath.flagKey, logPath.envKey)
	viper.BindEnv(membersLimit.flagKey, membersLimit.envKey)
	viper.BindEnv(playlistLimit.flagKey, playlistLimit.envKey)
	viper.BindEnv(redisPort.flagKey, redisPort.envKey)
	viper.BindEnv(redisHost.flagKey, redisHost.envKey)
	viper.BindEnv(redisPassword.flagKey, redisPassword.envKey)

	viper.SetDefault(secret.flagKey, secret.defaultValue)
	viper.SetDefault(port.flagKey, port.defaultValue)
	viper.SetDefault(host.flagKey, host.defaultValue)
	viper.SetDefault(logLevel.flagKey, logLevel.defaultValue)
	viper.SetDefault(logPath.flagKey, logPath.defaultValue)
	viper.SetDefault(membersLimit.flagKey, membersLimit.defaultValue)
	viper.SetDefault(playlistLimit.flagKey, playlistLimit.defaultValue)
	viper.SetDefault(redisPort.flagKey, redisPort.defaultValue)
	viper.SetDefault(redisHost.flagKey, redisHost.defaultValue)
	viper.SetDefault(redisPassword.flagKey, redisPassword.defaultValue)

	config := &app.AppConfig{
		Secret:        viper.GetString(secret.flagKey),
		Host:          viper.GetString(host.flagKey),
		Port:          viper.GetInt(port.flagKey),
		LogLevel:      viper.GetString(logLevel.flagKey),
		LogPath:       viper.GetString(logPath.flagKey),
		MembersLimit:  viper.GetInt(membersLimit.flagKey),
		PlaylistLimit: viper.GetInt(playlistLimit.flagKey),
		RedisPort:     viper.GetInt(redisPort.flagKey),
		RedisHost:     viper.GetString(redisHost.flagKey),
		RedisPassword: viper.GetString(redisPassword.flagKey),
	}

	return config
}

func main() {
	ctx := context.Background()

	appConfig := loadAppConfig()

	jsonConfig, _ := json.MarshalIndent(appConfig, "", "  ")
	fmt.Printf("starting app with config: %s\n", jsonConfig)

	log.Fatal(app.Run(ctx, appConfig))
}
