package sender

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	conf "github.com/ad/telegram-delete-join-messages/config"
	"github.com/go-telegram/bot"
	bm "github.com/go-telegram/bot/models"
)

type Sender struct {
	sync.RWMutex
	lgr              *slog.Logger
	config           *conf.Config
	Bot              *bot.Bot
	Config           *conf.Config
	deferredMessages map[int64]chan DeferredMessage
	lastMessageTimes map[int64]int64
}

func InitSender(lgr *slog.Logger, config *conf.Config) (*Sender, error) {
	sender := &Sender{
		lgr:              lgr,
		config:           config,
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
			// "message",
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

	go b.Start(context.Background())
	go sender.sendDeferredMessages()

	sender.Bot = b

	return sender, nil
}

func (s *Sender) handler(ctx context.Context, b *bot.Bot, update *bm.Update) {
	if s.config.Debug {
		s.lgr.Debug(formatUpdateForLog(update))
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
					Permissions: &bm.ChatPermissions{
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
