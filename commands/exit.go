package commands

import (
	"context"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Exit bot on /exit
func (c *Commands) Exit(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !slices.Contains(c.config.AllowedChatIDsList, update.Message.Chat.ID) {
		return
	}

	if slices.Contains(c.config.TelegramAdminIDsList, update.Message.From.ID) {
		fmt.Println("exiting... by", update.Message.From.ID)
		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "exiting...",
			ParseMode: models.ParseModeHTML,
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/exit): ", errSendMessage)
		}

		time.Sleep(1 * time.Second)

		os.Exit(0)
	}
}
