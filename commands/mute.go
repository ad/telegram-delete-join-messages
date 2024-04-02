package commands

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Mute user on /mute
func (c *Commands) Mute(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !slices.Contains(c.config.AllowedChatIDsList, update.Message.Chat.ID) {
		return
	}

	if update.Message.From.ID == c.config.TelegramAdminID {
		userID := update.Message.ReplyToMessage.From.ID
		chatID := update.Message.Chat.ID

		fmt.Println("muting", userID, "::", chatID, "by", update.Message.From.ID)

		_, errRestrictChatMember := b.RestrictChatMember(
			context.Background(),
			&bot.RestrictChatMemberParams{
				ChatID: chatID,
				UserID: userID,
				Permissions: &models.ChatPermissions{
					CanSendMessages:      false,
					CanSendAudios:        false,
					CanSendDocuments:     false,
					CanSendPhotos:        false,
					CanSendVideos:        false,
					CanSendPolls:         false,
					CanSendVideoNotes:    false,
					CanSendVoiceNotes:    false,
					CanSendOtherMessages: false,
				},
				UntilDate: int(time.Now().Add(365 * 24 * time.Hour).Unix()),
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
