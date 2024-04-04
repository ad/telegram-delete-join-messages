package commands

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Respond with the ID of the user who sent the message with /id
func (c *Commands) Id(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      fmt.Sprintf("Your ID is %d, chat id is %d", update.Message.From.ID, update.Message.Chat.ID),
		ParseMode: models.ParseModeHTML,
	})
}
