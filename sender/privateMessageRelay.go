package sender

import (
	"database/sql"
	"fmt"
	"slices"
	"strconv"

	"github.com/ad/telegram-delete-join-messages/data"
	"github.com/go-telegram/bot/models"
)

func (s *Sender) relayVerifiedPrivateMessageToAdmins(update *models.Update) {
	if update == nil || update.Message == nil || len(s.config.TelegramAdminIDsList) == 0 {
		return
	}

	message := update.Message
	if message.Chat.Type != "private" || message.From == nil || message.From.IsBot {
		return
	}

	if slices.Contains(s.config.TelegramAdminIDsList, message.Chat.ID) {
		return
	}

	vote, err := data.CheckVote(s.DB, message.Chat.ID, message.From.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Error(fmt.Sprintf("relayVerifiedPrivateMessageToAdmins CheckVote error: %s", err.Error()))
		return
	}

	if vote == 0 {
		return
	}

	metadata := fmt.Sprintf("📨 ЛС от пользователя\n\nID: %d\n%s\n\nНа это сообщение можно ответить и пользователь получит его анонимно", message.From.ID, buildData(message.From, vote))
	fromChatID := strconv.FormatInt(message.Chat.ID, 10)

	for _, adminID := range s.config.TelegramAdminIDsList {
		adminID := adminID

		s.MakeRequestDeferred(DeferredMessage{
			Method: "sendMessage",
			ChatID: adminID,
			Text:   metadata,
		}, s.SendResult)

		s.MakeRequestDeferred(DeferredMessage{
			Method:     "forwardMessage",
			ChatID:     adminID,
			fromChatID: fromChatID,
			messageID:  message.ID,
		}, func(result SendResult) error {
			if result.Error == nil && result.MessageID != 0 {
				s.storeForwardTarget(adminID, result.MessageID, message.From.ID)
			}

			return s.SendResult(result)
		})
	}
}
