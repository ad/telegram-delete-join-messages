package commands

import (
	"context"
	"fmt"
	"slices"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Unban user on /unban
func (c *Commands) Unban(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !slices.Contains(c.config.AllowedChatIDsList, update.Message.Chat.ID) {
		return
	}

	if slices.Contains(c.config.TelegramAdminIDsList, update.Message.From.ID) {
		userID := update.Message.ReplyToMessage.From.ID
		chatID := update.Message.Chat.ID

		fmt.Println("unbaning", userID, "::", chatID, "by", update.Message.From.ID)

		_, errUnbanChatMember := b.UnbanChatMember(
			context.Background(),
			&bot.UnbanChatMemberParams{
				ChatID: chatID,
				UserID: userID,
			},
		)

		if errUnbanChatMember != nil {
			fmt.Printf("Error unbanning member %d: %s\n", userID, errUnbanChatMember.Error())
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
