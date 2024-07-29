package sender

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/ad/telegram-delete-join-messages/commands"
	conf "github.com/ad/telegram-delete-join-messages/config"
	"github.com/ad/telegram-delete-join-messages/data"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Sender struct {
	sync.RWMutex
	lgr              *slog.Logger
	config           *conf.Config
	DB               *sql.DB
	Bot              *bot.Bot
	Config           *conf.Config
	deferredMessages map[int64]chan DeferredMessage
	lastMessageTimes map[int64]int64
	convHandler      *ConversationHandler
}

const (
	towerStage = iota // Definition of the first name stage = 0
	roomStage
)

func InitSender(lgr *slog.Logger, config *conf.Config, db *sql.DB) (*Sender, error) {
	command := commands.InitCommands(config)
	sender := &Sender{
		lgr:              lgr,
		config:           config,
		DB:               db,
		deferredMessages: make(map[int64]chan DeferredMessage),
		lastMessageTimes: make(map[int64]int64),
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(sender.handler),
		bot.WithSkipGetMe(),
		// list of alloweed updates
		// https://core.telegram.org/bots/api#update
		bot.WithAllowedUpdates(bot.AllowedUpdates{
			// "callback_query",
			// "channel_post",
			// "chat_boost",
			"chat_join_request",
			"chat_member",
			// "chat_migration",
			// "chosen_inline_result",
			// "edited_channel_post",
			// "edited_message",
			// "inline_query",
			"message",
			// "message_reaction",
			// "message_reaction_count",
			"my_chat_member",
			// "poll",
			// "poll_answer",
			// "removed_chat_boost",
		}),
	}

	// if config.Debug {
	// 	opts = append(opts, bot.WithDebug())
	// }

	b, newBotError := bot.New(config.TelegramToken, opts...)
	if newBotError != nil {
		return nil, fmt.Errorf("start bot error: %s", newBotError)
	}

	// Create a conversation handler and add stages
	convHandler := NewConversationHandler()
	convHandler.AddStage(towerStage, sender.towerHandler)
	convHandler.AddStage(roomStage, sender.roomHandler)

	sender.convHandler = convHandler

	go b.Start(context.Background())
	go sender.sendDeferredMessages()

	sender.Bot = b

	b.RegisterHandler(bot.HandlerTypeMessageText, "/kick", bot.MatchTypePrefix, command.Kick)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/id", bot.MatchTypePrefix, command.Id)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/mute", bot.MatchTypePrefix, command.Mute)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/unmute", bot.MatchTypePrefix, command.Unmute)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/ban", bot.MatchTypeExact, command.Ban)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/unban", bot.MatchTypeExact, command.Unban)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/exit", bot.MatchTypeExact, command.Exit)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/tldr", bot.MatchTypePrefix, command.TLDR)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, sender.startConversation)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/cancel", bot.MatchTypeExact, sender.cancelConversation)

	return sender, nil
}

func (s *Sender) handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if s.config.Debug {
		s.lgr.Debug(formatUpdateForLog(update))

		// call stage
		s.convHandler.CallStage(ctx, b, update)
	}

	if s.config.RestictOnJoin && update.Message != nil && update.Message.NewChatMembers != nil {
		s.lgr.Info(fmt.Sprintf("Restrict users %#v", update.Message.NewChatMembers))

		if !slices.Contains(s.config.AllowedChatIDsList, update.Message.Chat.ID) {
			s.lgr.Info(fmt.Sprintf("Chat ID %d is not in allowed list", update.Message.Chat.ID))
			return
		}

		for _, member := range update.Message.NewChatMembers {
			_, err := s.Bot.RestrictChatMember(
				context.Background(),
				&bot.RestrictChatMemberParams{
					ChatID: update.Message.Chat.ID,
					UserID: member.ID,
					Permissions: &models.ChatPermissions{
						CanSendMessages: false,
					},
					UntilDate: int(time.Now().Add(time.Second * time.Duration(s.config.RestrictOnJoinTime)).Unix()),
				},
			)
			if err != nil {
				s.lgr.Error(fmt.Sprintf("Error restricting member %d: %s", member.ID, err.Error()))
			}
		}
	}

	if s.config.DeleteJoinMessages && update.Message != nil && update.Message.NewChatMembers != nil {
		s.lgr.Info(fmt.Sprintf("Member joined %+v, chat ID %d", update.Message.NewChatMembers, update.Message.Chat.ID))

		if !slices.Contains(s.config.AllowedChatIDsList, update.Message.Chat.ID) {
			s.lgr.Info(fmt.Sprintf("Chat ID %d is not in allowed list", update.Message.Chat.ID))
			return
		}

		_, err := s.Bot.DeleteMessage(
			context.Background(),
			&bot.DeleteMessageParams{
				ChatID:    update.Message.Chat.ID,
				MessageID: update.Message.ID,
			},
		)

		if err != nil {
			s.lgr.Error(fmt.Sprintf("Error deleting message: %d, %d %s", update.Message.Chat.ID, update.Message.ID, err.Error()))
		}

		return
	}

	if s.config.DeleteLeaveMessages && update.Message != nil && update.Message.LeftChatMember != nil {
		s.lgr.Info(fmt.Sprintf("Member has left %+v, chat ID %d", update.Message.LeftChatMember, update.Message.Chat.ID))

		if !slices.Contains(s.config.AllowedChatIDsList, update.Message.Chat.ID) {
			s.lgr.Info(fmt.Sprintf("Chat ID %d is not in allowed list", update.Message.Chat.ID))
			return
		}

		_, err := s.Bot.DeleteMessage(
			context.Background(),
			&bot.DeleteMessageParams{
				ChatID:    update.Message.Chat.ID,
				MessageID: update.Message.ID,
			},
		)

		if err != nil {
			s.lgr.Error(fmt.Sprintf("Error deleting message %d, %d: %s", update.Message.Chat.ID, update.Message.ID, err.Error()))
		}

		return
	}
}

// Handle /start command to start getting the user's tower
func (s *Sender) startConversation(ctx context.Context, b *bot.Bot, update *models.Update) {
	// check room presense in db
	vote, err := data.CheckVote(s.DB, update.Message.Chat.ID, update.Message.From.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Info(fmt.Sprintf("startConversation CheckVote: %s", err.Error()))
	}

	if vote != "" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Вы уже голосовали: %s", vote),
		})

		return
	}

	s.convHandler.SetActiveStage(towerStage, int(update.Message.From.ID)) //start the tower stage

	// Ask user to enter their name
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Пожалуйста, скажите, из какой вы башни?",
	})
}

// Handle the tower stage to get the user's tower
func (s *Sender) towerHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	// firstName = update.Message.Text

	// check tower presense in db
	tower := update.Message.Text

	allowedTowers := []string{
		"1", "2",
		"б", "Б", "к", "К",
		"первой", "второй",
		"Первой", "Второй",
		"первого", "второго",
		"Первого", "Второго",
		"байконурская", "королева", "королёва",
		"Байконурская", "Королева", "Королёва",
	}

	if !slices.Contains(allowedTowers, tower) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Извините, но башни %s у нас нет. Попробуйте еще раз", tower),
		})

		return
	}

	s.convHandler.SetActiveStage(roomStage, int(update.Message.From.ID)) //change stage to last name stage
	// s.convHandler.End() // end the conversation

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Хорошо, из башни #%s. А теперь номер квартиры :)", tower),
	})
}

// Handle the room stage to get the user's room
func (s *Sender) roomHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	// check room presense in db
	room := update.Message.Text

	vote, err := data.CheckVote(s.DB, update.Message.Chat.ID, update.Message.From.ID)
	if err != nil && err != sql.ErrNoRows {
		s.lgr.Info(fmt.Sprintf("roomHandler CheckVote(%s): %s", room, err.Error()))

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Произошла ошибка при проверке голоса",
		})

		return
	}

	if vote != "" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Вы уже голосовали: %s", vote),
		})

		return
	}

	allowedRoomsMin := 1
	allowedRoomsMax := 344

	if roomInt, err := strconv.Atoi(room); err != nil || roomInt < allowedRoomsMin || roomInt > allowedRoomsMax {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Извините, но квартиры %s у нас нет. Попробуйте еще раз", room),
		})

		return
	}

	s.convHandler.End(int(update.Message.From.ID)) // end the conversation

	user_data := fmt.Sprintf("id %d %s %s %s", update.Message.From.ID, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.Username)

	err = data.AddVote(s.DB, update.Message.Chat.ID, update.Message.From.ID, room, user_data)
	if err != nil {
		s.lgr.Info(fmt.Sprintf("roomHandler AddVote (%s): %s", room, err.Error()))

		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Спасибо, записали #%s :)", room),
	})
}

// Handle /cancel command to end the conversation
func (s *Sender) cancelConversation(ctx context.Context, b *bot.Bot, update *models.Update) {
	s.convHandler.End(int(update.Message.From.ID)) // end the conversation

	// Send a message to indicate the conversation has been cancelled
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Дело ваше",
	})
}
