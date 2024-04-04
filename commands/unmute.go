package commands

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// // Unmute user on /unmute
func (c *Commands) Unmute(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !slices.Contains(c.config.AllowedChatIDsList, update.Message.Chat.ID) {
		return
	}

	if slices.Contains(c.config.TelegramAdminIDsList, update.Message.From.ID) {
		userID := update.Message.ReplyToMessage.From.ID
		chatID := update.Message.Chat.ID

		fmt.Println("unmuting", userID, "::", chatID, "by", update.Message.From.ID)

		_, errRestrictChatMember := b.RestrictChatMember(
			context.Background(),
			&bot.RestrictChatMemberParams{
				ChatID: chatID,
				UserID: userID,
				Permissions: &models.ChatPermissions{
					CanSendMessages: true,
				},
				UntilDate: int(time.Now().Add(1 * time.Second).Unix()),
			},
		)

		if errRestrictChatMember != nil {
			fmt.Printf("Error restricting member %d: %s\n", userID, errRestrictChatMember.Error())
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
