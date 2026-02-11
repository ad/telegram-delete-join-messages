package sender

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ad/telegram-delete-join-messages/data"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var adminNotificationPrefixes = []string{
	"📝 Новая заявка на вступление",
	"✅ Пользователь добавлен в группу",
	"🎉 Пользователь присоединился к группе",
	"👋 Пользователь вышел из группы",
	"📨 ЛС от пользователя",
}

var adminNotificationUserIDPattern = regexp.MustCompile(`(?m)^ID:\s*(-?\d+)\b`)

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
		} else {
			s.notifyAdminsJoinApprove(ctx, &update.ChatJoinRequest.From, chatID)
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

func (s *Sender) notifyAdminsJoinRequest(_ context.Context, user *models.User, _ int64) {
	if len(s.config.TelegramAdminIDsList) == 0 {
		return
	}

	vote, err := data.CheckVote(s.DB, user.ID, user.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Error(fmt.Sprintf("notifyAdminsJoinRequest CheckVote error: %s", err.Error()))
	}

	message := fmt.Sprintf("📝 Новая заявка на вступление\n\n"+
		"ID: %d\n%s",
		user.ID,
		buildData(user, vote),
	)

	for _, adminID := range s.config.TelegramAdminIDsList {
		s.MakeRequestDeferred(DeferredMessage{
			Method: "sendMessage",
			ChatID: adminID,
			Text:   message,
		}, s.SendResult)
	}
}

func (s *Sender) notifyAdminsJoinApprove(_ context.Context, user *models.User, _ int64) {
	if len(s.config.TelegramAdminIDsList) == 0 {
		return
	}

	vote, err := data.CheckVote(s.DB, user.ID, user.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Error(fmt.Sprintf("notifyAdminsJoinRequest CheckVote error: %s", err.Error()))
	}

	message := fmt.Sprintf("✅ Пользователь добавлен в группу\n\n"+
		"ID: %d\n%s",
		user.ID,
		buildData(user, vote),
	)

	for _, adminID := range s.config.TelegramAdminIDsList {
		s.MakeRequestDeferred(DeferredMessage{
			Method: "sendMessage",
			ChatID: adminID,
			Text:   message,
		}, s.SendResult)
	}
}

func (s *Sender) notifyAdminsUserJoined(_ context.Context, user *models.User, _ int64) {
	if len(s.config.TelegramAdminIDsList) == 0 {
		return
	}

	vote, err := data.CheckVote(s.DB, user.ID, user.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Error(fmt.Sprintf("notifyAdminsUserJoined CheckVote error: %s", err.Error()))
	}

	message := fmt.Sprintf("✅ Пользователь присоединился к группе\n\n"+
		"ID: %d\n%s",
		user.ID,
		buildData(user, vote),
	)

	for _, adminID := range s.config.TelegramAdminIDsList {
		s.MakeRequestDeferred(DeferredMessage{
			Method: "sendMessage",
			ChatID: adminID,
			Text:   message,
		}, s.SendResult)
	}
}

func (s *Sender) notifyAdminsUserLeft(_ context.Context, user *models.User, _ int64) {
	if len(s.config.TelegramAdminIDsList) == 0 {
		return
	}

	vote, err := data.CheckVote(s.DB, user.ID, user.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Error(fmt.Sprintf("notifyAdminsUserLeft CheckVote error: %s", err.Error()))
	}

	message := fmt.Sprintf("👋 Пользователь вышел из группы\n\n"+
		"ID: %d\n%s",
		user.ID,
		buildData(user, vote),
	)

	for _, adminID := range s.config.TelegramAdminIDsList {
		s.MakeRequestDeferred(DeferredMessage{
			Method: "sendMessage",
			ChatID: adminID,
			Text:   message,
		}, s.SendResult)
	}
}

func (s *Sender) notifyAdminsBotAddedToGroup(_ context.Context, chat *models.Chat) {
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

func buildData(user *models.User, vote int) string {
	usernameStr := ""

	if user.Username != "" {
		usernameStr = fmt.Sprintf("Username: @%s\n", user.Username)
	}

	nameStr := ""

	if user.FirstName != "" {
		nameStr = fmt.Sprintf("Имя: %s\n", user.FirstName)
	}

	surnameStr := ""

	if user.LastName != "" {
		surnameStr = fmt.Sprintf("Фамилия: %s\n", user.LastName)
	}

	voteStr := ""

	if vote > 0 {
		voteStr = fmt.Sprintf("Vote: %d\n", vote)
	}

	return fmt.Sprintf("%s%s%s%s", usernameStr, nameStr, surnameStr, voteStr)
}

func extractUserIDFromAdminNotification(message *models.Message) (int64, bool) {
	if message == nil {
		return 0, false
	}

	text := message.Text
	if text == "" {
		text = message.Caption
	}

	if text == "" {
		return 0, false
	}

	isAdminNotification := false
	for _, prefix := range adminNotificationPrefixes {
		if strings.HasPrefix(text, prefix) {
			isAdminNotification = true
			break
		}
	}

	if !isAdminNotification {
		return 0, false
	}

	matches := adminNotificationUserIDPattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return 0, false
	}

	userID, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, false
	}

	return userID, true
}
