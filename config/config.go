package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

const ConfigFileName = "/data/options.json"

// Config ...
type Config struct {
	TelegramToken string `json:"TELEGRAM_TOKEN"`
	TelegramAdmin string `json:"TELEGRAM_ADMIN_ID"`

	TelegramAdminID int64

	DeleteJoinMessages  bool `json:"DELETE_JOIN"`
	DeleteLeaveMessages bool `json:"DELETE_LEAVE"`

	RestictOnJoin      bool `json:"RESTRICT_ON_JOIN"`
	RestrictOnJoinTime int  `json:"RESTRICT_ON_JOIN_TIME"`

	AllowedChatIDs     string  `json:"ALLOWED_CHAT_IDS"`
	AllowedChatIDsList []int64 `json:"-"`

	YandexToken string `json:"YANDEX_TOKEN"`

	Debug bool `json:"DEBUG"`
}

func InitConfig(args []string) (*Config, error) {
	var config = &Config{
		TelegramToken:   "",
		TelegramAdmin:   "",
		TelegramAdminID: 0,

		DeleteJoinMessages:  false,
		DeleteLeaveMessages: false,

		RestictOnJoin:      true,
		RestrictOnJoinTime: 120,

		AllowedChatIDs:     "",
		AllowedChatIDsList: []int64{},

		YandexToken: "",

		Debug: false,
	}

	var initFromFile = false

	if _, err := os.Stat(ConfigFileName); err == nil {
		jsonFile, err := os.Open(ConfigFileName)
		if err == nil {
			byteValue, _ := io.ReadAll(jsonFile)
			if err = json.Unmarshal(byteValue, &config); err == nil {
				initFromFile = true
			} else {
				fmt.Printf("error on unmarshal config from file %s\n", err.Error())
			}
		}
	}

	if !initFromFile {
		flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
		flags.StringVar(&config.TelegramToken, "telegramToken", lookupEnvOrString("TELEGRAM_TOKEN", config.TelegramToken), "TELEGRAM_TOKEN")
		flags.StringVar(&config.TelegramAdmin, "telegramAdminID", lookupEnvOrString("TELEGRAM_ADMIN_ID", config.TelegramAdmin), "TELEGRAM_ADMIN_ID")

		flags.BoolVar(&config.DeleteJoinMessages, "deleteJoin", lookupEnvOrBool("DELETE_JOIN", config.DeleteJoinMessages), "DELETE_JOIN")
		flags.BoolVar(&config.DeleteLeaveMessages, "deleteLeave", lookupEnvOrBool("DELETE_LEAVE", config.DeleteLeaveMessages), "DELETE_LEAVE")

		flags.BoolVar(&config.RestictOnJoin, "restrictOnJoin", lookupEnvOrBool("RESTRICT_ON_JOIN", config.RestictOnJoin), "RESTRICT_ON_JOIN")
		flags.IntVar(&config.RestrictOnJoinTime, "restrictOnJoinTime", lookupEnvOrInt("RESTRICT_ON_JOIN_TIME", config.RestrictOnJoinTime), "RESTRICT_ON_JOIN_TIME")

		flags.StringVar(&config.AllowedChatIDs, "allowedChatIDs", lookupEnvOrString("ALLOWED_CHAT_IDS", config.AllowedChatIDs), "ALLOWED_CHAT_IDS")

		flags.StringVar(&config.YandexToken, "yandexToken", lookupEnvOrString("YANDEX_TOKEN", config.YandexToken), "YANDEX_TOKEN")

		flags.BoolVar(&config.Debug, "debug", lookupEnvOrBool("DEBUG", config.Debug), "Debug")

		if err := flags.Parse(args[1:]); err != nil {
			return nil, err
		}
	}

	if config.TelegramAdmin != "" {
		if chatID, err := strconv.ParseInt(config.TelegramAdmin, 10, 64); err == nil {
			config.TelegramAdminID = chatID
		}
	}

	if config.AllowedChatIDs != "" {
		chatIDS := strings.Split(config.AllowedChatIDs, ",")
		for _, chatID := range chatIDS {
			if chatIDInt, err := strconv.ParseInt(strings.Trim(chatID, "\n\t "), 10, 64); err == nil {
				config.AllowedChatIDsList = append(config.AllowedChatIDsList, chatIDInt)
			}
		}
	}

	return config, nil
}
