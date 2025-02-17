package sender

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"slices"
	"strconv"

	"github.com/ad/telegram-delete-join-messages/data"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	towerStage = iota // Definition of the first name stage = 0
	chairmanStage
	roomStage
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

// Handle /start command to start getting the user's tower
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

	s.convHandler.SetActiveStage(towerStage, int(update.Message.From.ID)) //start the tower stage

	// Ask user to enter their name
	_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "📝 Пожалуйста, ответьте на пару вопросов.\n\n🏬 В какой башне вы живете?",
	})

	if errSendMessage != nil {
		fmt.Println("errSendMessage (/start): ", errSendMessage)
	}
}

// Handle the tower stage to get the user's tower
func (s *Sender) towerHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// check if message is private
	if update.Message.Chat.Type != "private" {
		return
	}

	allowedTowers := []string{
		"1", "2",
		"б", "Б", "к", "К",
		"Башня 1", "Башня 2",
		"Башня Б", "Башня К",
		"Башня б", "Башня к",
		"башня б", "башня к",
		"первой", "второй",
		"Первой", "Второй",
		"первого", "второго",
		"Первого", "Второго",
		"байконурская", "королева", "королёва",
		"Байконурская", "Королева", "Королёва",
	}

	tower := update.Message.Text

	if !slices.Contains(allowedTowers, tower) {
		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Вы дали неправильный ответ.\nЕсли вы не знаете ответа, то вам сюда не надо.",
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/tower): ", errSendMessage)
		}

		return
	}

	s.convHandler.SetActiveStage(chairmanStage, int(update.Message.From.ID)) //change stage
	// s.convHandler.End() // end the conversation

	_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		// Text:   "✅ Хорошо, похоже мы соседи...\n\n👶 Как называется детский сад, который находится в нашем доме?",
		Text: "✅ Хорошо, похоже мы соседи...\n\n👴 А какая фамилия у нашего председателя?",
	})

	if errSendMessage != nil {
		fmt.Println("errSendMessage (/tower): ", errSendMessage)
	}
}

// Handle the zabava stage to get the user's zabava
func (s *Sender) zabavaHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// check if message is private
	if update.Message.Chat.Type != "private" {
		return
	}

	allowedTowers := []string{
		"забава", "Забава",
		"zabava", "Zabava",
		"Крошка", "крошка",
		"Кроха", "кроха",
		"забава сад", "Забава сад",
	}

	tower := update.Message.Text

	if !slices.Contains(allowedTowers, tower) {
		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Вы дали неправильный ответ. Если вы не знаете ответа, то вам сюда не надо.",
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/zabava): ", errSendMessage)
		}

		return
	}

	s.convHandler.SetActiveStage(roomStage, int(update.Message.From.ID)) //change stage to last name stage
	// s.convHandler.End() // end the conversation

	_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "✅ Хорошо, мы соседи.\n\n🚪 Назовите номер квартиры (не бойтесь, это просто проверка, чтобы быть уверенными, что вы не просто проходили мимо).",
	})

	if errSendMessage != nil {
		fmt.Println("errSendMessage (/zabava): ", errSendMessage)
	}
}

// Handle the chairman stage to get the user's chairman
func (s *Sender) chairmanHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// check if message is private
	if update.Message.Chat.Type != "private" {
		return
	}

	allowedTowers := []string{
		"пащенко", "Пащенко",
	}

	tower := update.Message.Text

	if !slices.Contains(allowedTowers, tower) {
		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Вы дали неправильный ответ. Если вы не знаете ответа, то вам сюда не надо.",
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/chairman): ", errSendMessage)
		}

		return
	}

	s.convHandler.SetActiveStage(roomStage, int(update.Message.From.ID)) //change stage to last name stage
	// s.convHandler.End() // end the conversation

	_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "✅ Хорошо, мы соседи.\n\n🚪 Назовите номер квартиры (не бойтесь, это просто проверка, чтобы быть уверенными, что вы не просто проходили мимо).",
	})

	if errSendMessage != nil {
		fmt.Println("errSendMessage (/chairman): ", errSendMessage)
	}
}

// Handle the room stage to get the user's room
func (s *Sender) roomHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// check if message is private
	if update.Message.Chat.Type != "private" {
		return
	}

	// check room presense in db
	room := update.Message.Text

	vote, err := data.CheckVote(s.DB, update.Message.Chat.ID, update.Message.From.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Info(fmt.Sprintf("roomHandler CheckVote(%s): %s", room, err.Error()))

		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Произошла ошибка при проверке ответа. Попробуйте еще раз",
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/room): ", errSendMessage)
		}

		return
	}

	if vote != 0 {
		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "✅ Вас уже записали",
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/room): ", errSendMessage)
		}

		return
	}

	allowedRoomsMin := 1
	allowedRoomsMax := 344

	if roomInt, err := strconv.Atoi(room); err != nil || roomInt < allowedRoomsMin || roomInt > allowedRoomsMax {
		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Извините, но такой квартиры у нас нет. Попробуйте еще раз, но после нескольких неправильных ответов вас заблокируют.",
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/room): ", errSendMessage)
		}

		return
	}

	s.convHandler.End(int(update.Message.From.ID)) // end the conversation

	user_data := fmt.Sprintf("id %d %s %s %s", update.Message.From.ID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.Username)

	err = data.AddVote(s.DB, update.Message.Chat.ID, update.Message.From.ID, room, user_data)
	if err != nil {
		s.lgr.Info(fmt.Sprintf("roomHandler AddVote (%s): %s", room, err.Error()))

		return
	}

	message := "🫶🏻 Спасибо, записали"

	if s.config.InviteLink != "" {
		message = message + "\n🤫 Теперь перейдите по ссылке: " + s.config.InviteLink
	}

	_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   message,
	})

	if errSendMessage != nil {
		fmt.Println("errSendMessage (/room): ", errSendMessage)
	}
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
