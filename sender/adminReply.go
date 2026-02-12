package sender

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var telegramMessageLinkPattern = regexp.MustCompile(`(?i)(?:https?://)?(?:t\.me|telegram\.me)/[^\s]+|tg://privatepost\?[^\s]+`)

type groupReplyTarget struct {
	chatID          int64
	messageID       int
	messageThreadID int
}

var (
	errNotTelegramMessageLink = errors.New("not a telegram message link")
	errInvalidTelegramLink    = errors.New("invalid telegram link")
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
		if target, ok, parseErr := s.resolveGroupReplyTargetFromReference(ctx, b, message.ReplyToMessage); ok {
			_, err := b.CopyMessage(ctx, &bot.CopyMessageParams{
				ChatID:          target.chatID,
				FromChatID:      message.Chat.ID,
				MessageID:       message.ID,
				MessageThreadID: target.messageThreadID,
				ReplyParameters: &models.ReplyParameters{MessageID: target.messageID},
			})
			if err != nil {
				s.lgr.Error(fmt.Sprintf(
					"admin group reply relay failed for admin=%d target_chat=%d target_message=%d: %s",
					message.Chat.ID,
					target.chatID,
					target.messageID,
					err.Error(),
				))

				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: message.Chat.ID,
					Text: fmt.Sprintf(
						"Не удалось отправить ответ в группу (chat_id=%d, message_id=%d)",
						target.chatID,
						target.messageID,
					),
				})

				return true
			}

			return true
		} else if parseErr != "" {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: message.Chat.ID,
				Text:   parseErr,
			})
			return true
		}

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

func (s *Sender) resolveGroupReplyTargetFromReference(ctx context.Context, b *bot.Bot, reference *models.Message) (groupReplyTarget, bool, string) {
	if reference == nil {
		return groupReplyTarget{}, false, ""
	}

	for _, candidate := range extractTelegramMessageLinks(reference) {
		target, err := parseTelegramMessageTargetFromLink(candidate, func(username string) (int64, error) {
			chat, err := b.GetChat(ctx, &bot.GetChatParams{
				ChatID: "@" + username,
			})
			if err != nil {
				return 0, err
			}

			return chat.ID, nil
		})

		if err == nil {
			return target, true, ""
		}

		if errors.Is(err, errNotTelegramMessageLink) {
			continue
		}

		if errors.Is(err, errInvalidTelegramLink) {
			return groupReplyTarget{}, false, "Не удалось разобрать ссылку на сообщение. Используйте ссылку вида https://t.me/c/<chat>/<message> или https://t.me/<username>/<message>."
		}

		return groupReplyTarget{}, false, fmt.Sprintf("Не удалось определить группу по ссылке: %s", err.Error())
	}

	return groupReplyTarget{}, false, ""
}

func extractTelegramMessageLinks(message *models.Message) []string {
	if message == nil {
		return nil
	}

	links := map[string]struct{}{}
	result := []string{}

	addLink := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}

		if _, exists := links[raw]; exists {
			return
		}

		links[raw] = struct{}{}
		result = append(result, raw)
	}

	appendFromEntities := func(entities []models.MessageEntity) {
		for _, entity := range entities {
			if entity.Type == models.MessageEntityTypeTextLink && entity.URL != "" {
				addLink(entity.URL)
			}
		}
	}

	appendFromText := func(text string) {
		if text == "" {
			return
		}

		matches := telegramMessageLinkPattern.FindAllString(text, -1)
		for _, match := range matches {
			addLink(cleanTelegramLinkCandidate(match))
		}
	}

	appendFromEntities(message.Entities)
	appendFromEntities(message.CaptionEntities)
	appendFromText(message.Text)
	appendFromText(message.Caption)

	return result
}

func cleanTelegramLinkCandidate(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), ".,!?:;)]}")
}

func parseTelegramMessageTargetFromLink(raw string, resolveUsername func(username string) (int64, error)) (groupReplyTarget, error) {
	link, err := parseNormalizedTelegramURL(raw)
	if err != nil {
		return groupReplyTarget{}, err
	}

	if strings.EqualFold(link.Scheme, "tg") {
		return parseTGPrivatePostTarget(link)
	}

	host := strings.ToLower(strings.TrimPrefix(link.Host, "www."))
	if host != "t.me" && host != "telegram.me" {
		return groupReplyTarget{}, errNotTelegramMessageLink
	}

	parts := strings.Split(strings.Trim(link.Path, "/"), "/")
	if len(parts) < 2 {
		return groupReplyTarget{}, errInvalidTelegramLink
	}

	if parts[0] == "c" {
		return parseInternalChatLink(parts, link.Query())
	}

	username := parts[0]
	if !isValidTelegramUsername(username) {
		return groupReplyTarget{}, errInvalidTelegramLink
	}

	target, err := parsePublicChatLink(parts, link.Query())
	if err != nil {
		return groupReplyTarget{}, err
	}

	chatID, err := resolveUsername(username)
	if err != nil {
		return groupReplyTarget{}, err
	}

	target.chatID = chatID
	return target, nil
}

func parseNormalizedTelegramURL(raw string) (*url.URL, error) {
	raw = cleanTelegramLinkCandidate(raw)
	if raw == "" {
		return nil, errInvalidTelegramLink
	}

	normalized := raw
	if !strings.Contains(normalized, "://") && strings.HasPrefix(strings.ToLower(normalized), "t.me/") {
		normalized = "https://" + normalized
	}

	link, err := url.Parse(normalized)
	if err != nil {
		return nil, errInvalidTelegramLink
	}

	return link, nil
}

func parseInternalChatLink(parts []string, query url.Values) (groupReplyTarget, error) {
	if len(parts) < 3 {
		return groupReplyTarget{}, errInvalidTelegramLink
	}

	chatID, err := parseInternalChatID(parts[1])
	if err != nil {
		return groupReplyTarget{}, err
	}

	messageID, err := parsePositiveInt(parts[len(parts)-1])
	if err != nil {
		return groupReplyTarget{}, errInvalidTelegramLink
	}

	return groupReplyTarget{
		chatID:          chatID,
		messageID:       messageID,
		messageThreadID: extractThreadID(parts, query),
	}, nil
}

func parsePublicChatLink(parts []string, query url.Values) (groupReplyTarget, error) {
	if len(parts) < 2 {
		return groupReplyTarget{}, errInvalidTelegramLink
	}

	messageID, err := parsePositiveInt(parts[len(parts)-1])
	if err != nil {
		return groupReplyTarget{}, errInvalidTelegramLink
	}

	return groupReplyTarget{
		messageID:       messageID,
		messageThreadID: extractThreadID(parts, query),
	}, nil
}

func parseTGPrivatePostTarget(link *url.URL) (groupReplyTarget, error) {
	if !strings.EqualFold(link.Host, "privatepost") {
		return groupReplyTarget{}, errNotTelegramMessageLink
	}

	query := link.Query()

	channelRaw := query.Get("channel")
	if channelRaw == "" {
		return groupReplyTarget{}, errInvalidTelegramLink
	}

	chatID, err := strconv.ParseInt(channelRaw, 10, 64)
	if err != nil {
		return groupReplyTarget{}, errInvalidTelegramLink
	}

	messageID, err := parsePositiveInt(query.Get("post"))
	if err != nil {
		return groupReplyTarget{}, errInvalidTelegramLink
	}

	return groupReplyTarget{
		chatID:          chatID,
		messageID:       messageID,
		messageThreadID: extractThreadID(nil, query),
	}, nil
}

func parseInternalChatID(raw string) (int64, error) {
	if raw == "" {
		return 0, errInvalidTelegramLink
	}

	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, errInvalidTelegramLink
		}
	}

	chatID, err := strconv.ParseInt("-100"+raw, 10, 64)
	if err != nil {
		return 0, errInvalidTelegramLink
	}

	return chatID, nil
}

func parsePositiveInt(raw string) (int, error) {
	if raw == "" {
		return 0, errInvalidTelegramLink
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, errInvalidTelegramLink
	}

	return value, nil
}

func extractThreadID(parts []string, query url.Values) int {
	if len(parts) >= 4 {
		if value, err := parsePositiveInt(parts[len(parts)-2]); err == nil {
			return normalizeMessageThreadID(value)
		}
	}

	for _, key := range []string{"thread", "topic"} {
		if value, err := parsePositiveInt(query.Get(key)); err == nil {
			return normalizeMessageThreadID(value)
		}
	}

	return 0
}

func normalizeMessageThreadID(threadID int) int {
	// In forum supergroups, links from General can contain topic/thread=1,
	// but Bot API methods expect no MessageThreadID for General.
	if threadID == 1 {
		return 0
	}

	return threadID
}

func isValidTelegramUsername(username string) bool {
	if len(username) < 5 || len(username) > 32 {
		return false
	}

	for _, ch := range username {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case ch == '_':
		default:
			return false
		}
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
