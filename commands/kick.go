package commands

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Kick user on /kick
func (c *Commands) Kick(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !slices.Contains(c.config.AllowedChatIDsList, update.Message.Chat.ID) {
		return
	}

	if update.Message.From.ID == c.config.TelegramAdminID {
		userID := update.Message.ReplyToMessage.From.ID
		chatID := update.Message.Chat.ID

		fmt.Println("kicking", userID, "::", chatID, "by", update.Message.From.ID)

		_, errRestrictChatMember := b.RestrictChatMember(
			context.Background(),
			&bot.RestrictChatMemberParams{
				ChatID: chatID,
				UserID: userID,
				Permissions: &models.ChatPermissions{
					CanSendMessages: false,
				},
				UntilDate: int(time.Now().Add(60 * time.Second).Unix()),
			},
		)

		if errRestrictChatMember != nil {
			fmt.Printf("Error restricting member %d: %s\n", userID, errRestrictChatMember.Error())

			return
		}

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

		_, errUnbanChatMember := b.UnbanChatMember(
			context.Background(),
			&bot.UnbanChatMemberParams{
				ChatID: chatID,
				UserID: userID,
			},
		)

		if errUnbanChatMember != nil {
			fmt.Printf("Error unbanning member %d: %s\n", userID, errUnbanChatMember.Error())

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
