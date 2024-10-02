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
	fmt.Println(formatUpdateForLog(update), update.ChatJoinRequest.Bio)

	chatID := update.ChatJoinRequest.Chat.ID
	fromID := update.ChatJoinRequest.From.ID

	vote, err := data.CheckVote(s.DB, fromID, fromID)
	if err != nil && err != sql.ErrNoRows {
		return
	}

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(formatUpdateForLog(update), "room number", vote)

	if vote != 0 {
		// TODO: add ban check
		_, errApproveChatJoinRequest := b.ApproveChatJoinRequest(
			ctx,
			&bot.ApproveChatJoinRequestParams{
				ChatID: chatID,
				UserID: fromID,
			},
		)

		if errApproveChatJoinRequest != nil {
			fmt.Println("errApproveChatJoinRequest: ", errApproveChatJoinRequest, "for", fromID)
		}

		return
	}

	s.convHandler.SetActiveStage(towerStage, int(fromID)) //start the tower stage

	_, errSendMessage := b.SendMessage(
		ctx,
		&bot.SendMessageParams{
			ChatID: fromID,
			Text:   "❓ Для входа в группу ответьте на пару вопросов.\n\n🏬 В какой башне вы живете?",
		},
	)

	if errSendMessage != nil {
		fmt.Println("errSendMessage: ", errSendMessage, "for", fromID)
	}

	_, errDeclineChatJoinRequest := b.DeclineChatJoinRequest(
		ctx,
		&bot.DeclineChatJoinRequestParams{
			ChatID: chatID,
			UserID: fromID,
		},
	)

	if errDeclineChatJoinRequest != nil {
		fmt.Println("errDeclineChatJoinRequest: ", errDeclineChatJoinRequest, "for", fromID)
	}

	fmt.Println("user join request declined", fromID)
}
