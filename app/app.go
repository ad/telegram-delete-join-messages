package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	conf "github.com/ad/telegram-delete-join-messages/config"
	"github.com/ad/telegram-delete-join-messages/logger"
	sndr "github.com/ad/telegram-delete-join-messages/sender"
)

var (
	config *conf.Config
)

func Run(ctx context.Context, w io.Writer, args []string) error {
	confLoad, errInitConfig := conf.InitConfig(os.Args)
	if errInitConfig != nil {
		return errInitConfig
	}

	config = confLoad

	lgr := logger.InitLogger(config.Debug)

	// Recovery
	defer func() {
		if p := recover(); p != nil {
			lgr.Error(fmt.Sprintf("panic recovered: %s; stack trace: %s", p, string(debug.Stack())))
		}
	}()

	sender, errInitSender := sndr.InitSender(lgr, config)
	if errInitSender != nil {
		return errInitSender
	}

	if len(config.TelegramAdminIDsList) != 0 {
		sender.MakeRequestDeferred(sndr.DeferredMessage{
			Method: "sendMessage",
			ChatID: config.TelegramAdminIDsList[0],
			Text:   "Bot restarted",
		}, sender.SendResult)
	}

	return nil
}
