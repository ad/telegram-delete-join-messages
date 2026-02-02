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

	go s.notifyAdminsJoinRequest(ctx, &update.ChatJoinRequest.From, chatID)

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

	s.convHandler.SetActiveStage(0, int(fromID)) //start conversation

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

func (s *Sender) notifyAdminsJoinRequest(ctx context.Context, user *models.User, chatID int64) {
	if len(s.config.TelegramAdminIDsList) == 0 {
		return
	}

	vote, err := data.CheckVote(s.DB, user.ID, user.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Error(fmt.Sprintf("notifyAdminsJoinRequest CheckVote error: %s", err.Error()))
	}

	message := fmt.Sprintf("📝 Новая заявка на вступление\n\n"+
		"ID: %d\n"+
		"Username: @%s\n"+
		"Имя: %s\n"+
		"Фамилия: %s\n"+
		"Vote: %d",
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		vote,
	)

	for _, adminID := range s.config.TelegramAdminIDsList {
		s.MakeRequestDeferred(DeferredMessage{
			Method: "sendMessage",
			ChatID: adminID,
			Text:   message,
		}, s.SendResult)
	}
}

func (s *Sender) notifyAdminsUserJoined(ctx context.Context, user *models.User, chatID int64) {
	if len(s.config.TelegramAdminIDsList) == 0 {
		return
	}

	vote, err := data.CheckVote(s.DB, user.ID, user.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Error(fmt.Sprintf("notifyAdminsUserJoined CheckVote error: %s", err.Error()))
	}

	message := fmt.Sprintf("✅ Пользователь присоединился к группе\n\n"+
		"ID: %d\n"+
		"Username: @%s\n"+
		"Имя: %s\n"+
		"Фамилия: %s\n"+
		"Vote: %d",
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		vote,
	)

	for _, adminID := range s.config.TelegramAdminIDsList {
		s.MakeRequestDeferred(DeferredMessage{
			Method: "sendMessage",
			ChatID: adminID,
			Text:   message,
		}, s.SendResult)
	}
}

func (s *Sender) notifyAdminsUserLeft(ctx context.Context, user *models.User, chatID int64) {
	if len(s.config.TelegramAdminIDsList) == 0 {
		return
	}

	vote, err := data.CheckVote(s.DB, user.ID, user.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Error(fmt.Sprintf("notifyAdminsUserLeft CheckVote error: %s", err.Error()))
	}

	message := fmt.Sprintf("👋 Пользователь вышел из группы\n\n"+
		"ID: %d\n"+
		"Username: @%s\n"+
		"Имя: %s\n"+
		"Фамилия: %s\n"+
		"Vote: %d",
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		vote,
	)

	for _, adminID := range s.config.TelegramAdminIDsList {
		s.MakeRequestDeferred(DeferredMessage{
			Method: "sendMessage",
			ChatID: adminID,
			Text:   message,
		}, s.SendResult)
	}
}

func (s *Sender) notifyAdminsBotAddedToGroup(ctx context.Context, chat *models.Chat) {
	if len(s.config.TelegramAdminIDsList) == 0 {
		return
	}

	forumEnabled := "Нет"
	if chat.IsForum {
		forumEnabled = "Да"
	}

	message := fmt.Sprintf("🤖 Бот добавлен в группу\n\n"+
		"ID: %d\n"+
		"Название: %s\n"+
		"Форум: %s",
		chat.ID,
		chat.Title,
		forumEnabled,
	)

	for _, adminID := range s.config.TelegramAdminIDsList {
		s.MakeRequestDeferred(DeferredMessage{
			Method: "sendMessage",
			ChatID: adminID,
			Text:   message,
		}, s.SendResult)
	}
}
