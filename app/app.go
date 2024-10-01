package app

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"runtime/debug"
	"strings"

	conf "github.com/ad/telegram-delete-join-messages/config"
	data "github.com/ad/telegram-delete-join-messages/data"
	"github.com/ad/telegram-delete-join-messages/logger"
	sndr "github.com/ad/telegram-delete-join-messages/sender"
)

func Run(ctx context.Context, w io.Writer, args []string) error {
	config, errInitConfig := conf.InitConfig(args)
	if errInitConfig != nil {
		return errInitConfig
	}

	lgr := logger.InitLogger(config.Debug)

	// Recovery
	defer func() {
		if p := recover(); p != nil {
			lgr.Error(fmt.Sprintf("panic recovered: %s; stack trace: %s", p, string(debug.Stack())))
		}
	}()

	var db *sql.DB

	lgr.Debug(fmt.Sprintf("DB_PATH: %s", config.DB_PATH))

	if strings.HasSuffix(config.DB_PATH, ".db") {
		dbSqlite, errInitSqliteDB := data.InitSqliteDB(config.DB_PATH)
		if errInitSqliteDB != nil {
			return errInitSqliteDB
		}

		db = dbSqlite
	}

	if strings.HasPrefix(config.DB_PATH, "postgres://") {
		dbPostgres, errInitPostgresDB := data.InitPostgresDB(config.DB_PATH)
		if errInitPostgresDB != nil {
			return errInitPostgresDB
		}

		db = dbPostgres
	}

	sender, errInitSender := sndr.InitSender(lgr, config, db)
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
