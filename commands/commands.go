package commands

import (
	conf "github.com/ad/telegram-delete-join-messages/config"
)

type Commands struct {
	config *conf.Config
}

func InitCommands(config *conf.Config) *Commands {
	return &Commands{
		config: config,
	}
}
