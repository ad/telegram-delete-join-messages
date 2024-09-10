package sender

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ad/telegram-delete-join-messages/data"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (s *Sender) HandleChatJoinRequest(ctx context.Context, b *bot.Bot, update *models.Update) {
	fmt.Println("chat join request for", update.ChatJoinRequest.From.ID, "from", update.ChatJoinRequest.Chat.ID)

	vote, err := data.CheckVote(s.DB, update.ChatJoinRequest.From.ID, update.ChatJoinRequest.From.ID)
	if err != nil && err != sql.ErrNoRows {
		return
	}

	fmt.Println(err)
	fmt.Println(vote)
	if vote != 0 {
		// TODO: add ban check
		_, errApproveChatJoinRequest := b.ApproveChatJoinRequest(
			ctx,
			&bot.ApproveChatJoinRequestParams{
				ChatID: update.ChatJoinRequest.Chat.ID,
				UserID: update.ChatJoinRequest.From.ID,
			},
		)

		if errApproveChatJoinRequest != nil {
			fmt.Println("errApproveChatJoinRequest: ", errApproveChatJoinRequest, "for", update.ChatJoinRequest.From.ID)
		}

		return
	}

	_, errDeclineChatJoinRequest := b.DeclineChatJoinRequest(
		ctx,
		&bot.DeclineChatJoinRequestParams{
			ChatID: update.ChatJoinRequest.Chat.ID,
			UserID: update.ChatJoinRequest.From.ID,
		},
	)

	if errDeclineChatJoinRequest != nil {
		fmt.Println("errDeclineChatJoinRequest: ", errDeclineChatJoinRequest, "for", update.ChatJoinRequest.From.ID)
	}

	fmt.Println("user join request declined", update.ChatJoinRequest.From.ID)
}
