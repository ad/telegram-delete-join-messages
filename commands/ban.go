package commands

import (
	"context"
	"fmt"
	"slices"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Ban user on /ban
func (c *Commands) Ban(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !slices.Contains(c.config.AllowedChatIDsList, update.Message.Chat.ID) {
		return
	}

	if slices.Contains(c.config.TelegramAdminIDsList, update.Message.From.ID) {
		userID := update.Message.ReplyToMessage.From.ID
		chatID := update.Message.Chat.ID

		fmt.Println("baning", userID, "::", chatID, "by", update.Message.From.ID)

		_, errBanChatMember := b.BanChatMember(
			context.Background(),
			&bot.BanChatMemberParams{
				ChatID: chatID,
				UserID: userID,
			},
		)

		if errBanChatMember != nil {
			fmt.Printf("Error banning member %d: %s\n", userID, errBanChatMember.Error())

			return
		}
	}

	_, err := b.DeleteMessage(
		context.Background(),
		&bot.DeleteMessageParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: update.Message.ID,
		},
	)

	if err != nil {
		fmt.Printf("Error deleting message %d, %d: %s\n", update.Message.Chat.ID, update.Message.ID, err.Error())
	}
}
