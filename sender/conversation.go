package sender

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/ad/telegram-delete-join-messages/config"
	"github.com/ad/telegram-delete-join-messages/data"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	UserBadAnswer = "❌ Вы дали неправильный ответ.\nЕсли вы не знаете ответа, то вам сюда не надо."
)

// ConversationHandler is a structure that manages conversation functions.
type ConversationHandler struct {
	active         map[int]bool            // a flag indicating whether the conversation is active
	currentStageId map[int]int             // the identifier of the active conversation stage
	stages         map[int]bot.HandlerFunc // a map of conversation stages
}

// NewConversationHandler returns a new instance of ConversationHandler.
func NewConversationHandler() *ConversationHandler {
	return &ConversationHandler{
		active:         make(map[int]bool),
		currentStageId: make(map[int]int),
		stages:         make(map[int]bot.HandlerFunc),
	}
}

// AddStage adds a conversation stage to the ConversationHandler.
func (c *ConversationHandler) AddStage(stageId int, hf bot.HandlerFunc) {
	c.stages[stageId] = hf
}

// SetActiveStage sets the active conversation stage.
// Invalid currentStageId is not checked because if the CallStage function encounters an invalid id,
// it will not process it, so the stageId is not checked.
// if stageId <= len(c.stages)
func (c *ConversationHandler) SetActiveStage(stageId int, userID int) {
	if a, ok := c.active[userID]; !ok || !a {
		c.active[userID] = true
	}

	c.currentStageId[userID] = stageId
}

func (c *ConversationHandler) GetActiveStage(userID int) int {
	if _, ok := c.active[userID]; ok {
		return c.currentStageId[userID]
	}

	return 0
}

func (c *ConversationHandler) GetStagesCount() int {
	return len(c.stages)
}

// CallStage calls the function of the active conversation stage.
func (c *ConversationHandler) CallStage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// check if message is private
	if update.Message.Chat.Type != "private" {
		return
	}

	if _, ok := c.active[int(update.Message.From.ID)]; ok {
		// hf = HandlerFunction
		if hf, ok := c.stages[c.currentStageId[int(update.Message.From.ID)]]; ok {
			hf(ctx, b, update)
		} else {
			log.Println("Error: Invalid stage id. No matching function found for the current stage id.")
		}
	}
}

// End ends the conversation.
func (c *ConversationHandler) End(userID int) {
	c.active[userID] = false
}

// Handle /start command to start conversation
func (s *Sender) startConversation(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// check if message is private
	if update.Message.Chat.Type != "private" {
		return
	}

	// check room presense in db
	vote, err := data.CheckVote(s.DB, update.Message.Chat.ID, update.Message.From.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Info(fmt.Sprintf("startConversation CheckVote: %s", err.Error()))
	}

	if vote != 0 {
		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "✅ Вас уже записали",
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/start): ", errSendMessage)
		}

		return
	}

	s.convHandler.SetActiveStage(0, int(update.Message.From.ID)) //start conversation

	// Ask user to enter their name
	_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "📝 Пожалуйста, ответьте на пару вопросов.\n\n🏬 В какой башне вы живете?",
	})

	if errSendMessage != nil {
		fmt.Println("errSendMessage (/start): ", errSendMessage)
	}
}

func (s *Sender) GetConversationById(index int) (*config.Conversation, error) {
	conversations := s.config.Conversations

	if index < 0 || index >= len(conversations) {
		return nil, fmt.Errorf("index out of range")
	}

	return &conversations[index], nil
}

// Handle stages
func (s *Sender) stageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// check if message is private
	if update.Message.Chat.Type != "private" {
		return
	}

	currentStageId := s.convHandler.GetActiveStage(int(update.Message.From.ID))

	conversation, err := s.GetConversationById(currentStageId)
	if err != nil {
		fmt.Println("errGetConversation (/stageHandler): ", err)
		return
	}

	// split conversation.variants by comma
	variants := strings.Split(conversation.Variants, ",")
	for i := range variants {
		variants[i] = strings.ToUpper(strings.TrimSpace(variants[i]))
	}

	userAnswer := strings.TrimSpace(update.Message.Text)

	if !slices.Contains(variants, strings.ToUpper(userAnswer)) {
		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   UserBadAnswer,
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/tower): ", errSendMessage)
		}

		return
	}

	stagesCount := NewConversationHandler().GetStagesCount()

	if currentStageId+1 >= stagesCount {
		result := s.lastStep(ctx, b, update, userAnswer, conversation.Answer)
		if result {
			s.convHandler.End(int(update.Message.From.ID)) // end the conversation
		}
	} else {
		s.convHandler.SetActiveStage(currentStageId+1, int(update.Message.From.ID)) //change stage

		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   conversation.Answer,
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/tower): ", errSendMessage)
		}
	}
}

// Handle the room stage to get the user's room
func (s *Sender) lastStep(ctx context.Context, b *bot.Bot, update *models.Update, userInput, answer string) bool {
	_, err := s.GetVoteFromDBForUser(ctx, b, update.Message.Chat.ID, update.Message.From.ID)
	if err != nil {
		s.lgr.Info(fmt.Sprintf("roomHandler GetVoteFromDBForUser (%s): %s", userInput, err.Error()))

		return false
	}

	user_data := fmt.Sprintf("id %d %s %s %s", update.Message.From.ID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.Username)

	err = data.AddVote(s.DB, update.Message.Chat.ID, update.Message.From.ID, userInput, user_data)
	if err != nil {
		s.lgr.Info(fmt.Sprintf("roomHandler AddVote (%s): %s", userInput, err.Error()))

		return false
	}

	if s.config.InviteLink != "" {
		answer = answer + "\n🤫 Теперь перейдите по ссылке: " + s.config.InviteLink
	}

	_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   answer,
	})

	if errSendMessage != nil {
		fmt.Println("errSendMessage (/room): ", errSendMessage)
	}

	return true
}

func (s *Sender) GetVoteFromDBForUser(ctx context.Context, b *bot.Bot, chatID, userID int64) (int, error) {
	vote, err := data.CheckVote(s.DB, chatID, userID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Info(fmt.Sprintf("roomHandler CheckVote: %s", err.Error()))

		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Произошла ошибка при проверке ответа. Попробуйте еще раз",
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/room): ", errSendMessage)
		}

		return 0, err
	}

	if vote != 0 {
		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "✅ Вас уже записали",
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/room): ", errSendMessage)
		}

		return vote, err
	}

	return 0, err
}

// Handle /cancel command to end the conversation
func (s *Sender) cancelConversation(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// check if message is private
	if update.Message.Chat.Type != "private" {
		return
	}

	s.convHandler.End(int(update.Message.From.ID)) // end the conversation

	// Send a message to indicate the conversation has been cancelled
	_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "🥺 Дело ваше, может быть в следующий раз",
	})

	if errSendMessage != nil {
		fmt.Println("errSendMessage (/cancel): ", errSendMessage)
	}
}
