package sender

import (
	"context"
	"fmt"
	"slices"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (s *Sender) handleAdminReplyToNotification(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update == nil || update.Message == nil {
		return false
	}

	message := update.Message
	if !slices.Contains(s.config.TelegramAdminIDsList, message.Chat.ID) {
		return false
	}

	if message.ReplyToMessage == nil {
		return false
	}

	targetUserID, ok := extractTargetUserIDFromAdminReply(message.ReplyToMessage)
	if !ok {
		targetUserID, ok = s.getForwardTarget(message.Chat.ID, int64(message.ReplyToMessage.ID))
	}
	if !ok {
		return false
	}

	if message.From != nil && targetUserID == message.From.ID {
		return false
	}

	_, err := b.CopyMessage(ctx, &bot.CopyMessageParams{
		ChatID:     targetUserID,
		FromChatID: message.Chat.ID,
		MessageID:  message.ID,
	})
	if err != nil {
		s.lgr.Error(fmt.Sprintf("admin reply relay failed for admin=%d user=%d: %s", message.Chat.ID, targetUserID, err.Error()))

		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: message.Chat.ID,
			Text:   fmt.Sprintf("Не удалось отправить сообщение пользователю ID %d", targetUserID),
		})

		return true
	}

	return true
}

func extractTargetUserIDFromAdminReply(replyToMessage *models.Message) (int64, bool) {
	if userID, ok := extractUserIDFromAdminNotification(replyToMessage); ok {
		return userID, true
	}

	if replyToMessage == nil || replyToMessage.ForwardOrigin == nil || replyToMessage.ForwardOrigin.MessageOriginUser == nil {
		return 0, false
	}

	return replyToMessage.ForwardOrigin.MessageOriginUser.SenderUser.ID, true
}
