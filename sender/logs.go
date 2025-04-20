package sender

import (
	"fmt"
	"strings"

	bm "github.com/go-telegram/bot/models"
)

func formatUpdateForLog(message *bm.Update) string {
	switch {
	case message.Message != nil:
		return formatMessageForLog(message)
	case message.EditedMessage != nil:
		return formatMEForLog(message)
	case message.ChannelPost != nil:
		return formatChannelpostForLog(message)
	case message.MessageReaction != nil:
		return formatMessageReactionForLog(message)
	case message.MessageReactionCount != nil:
		return formatMessageReactionCountForLog(message)
	case message.CallbackQuery != nil:
		return formatCallbackQueryForLog(message)
	case message.ChatJoinRequest != nil:
		return formatChatJoinRequestForLog(message)
	}
	// jsonData, _ := json.Marshal(message)
	// s.lgr.Debug(fmt.Sprintf("Message %s", string(jsonData)))

	return fmt.Sprintf("%+v", message)
}

func formatMessageForLog(message *bm.Update) string {
	if message.Message.Chat.Type == "private" {
		if message.Message.Caption != "" {
			return fmt.Sprintf(
				"PM from %d (%s): Photo with message: %s",
				message.Message.From.ID,
				getUserDataFromMessage(message.Message.From),
				message.Message.Caption,
			)
		}

		return fmt.Sprintf("PM from %d (%s): %s", message.Message.Chat.ID, getChatDataFromMessage(&message.Message.Chat), message.Message.Text)
	}

	if message.Message.Caption != "" {
		return fmt.Sprintf(
			"M from %d (%s): Photo with message: %s",
			message.Message.From.ID,
			getUserDataFromMessage(message.Message.From),
			message.Message.Caption,
		)
	}

	if message.Message.ForumTopicCreated != nil {
		return fmt.Sprintf(
			"M from %d (%s): Forum topic %q created",
			message.Message.From.ID,
			getUserDataFromMessage(message.Message.From),
			message.Message.ReplyToMessage.ForumTopicCreated.Name,
		)
	}

	if message.Message.ForumTopicReopened != nil {
		return fmt.Sprintf(
			"M from %d (%s): Forum topic reopened",
			message.Message.From.ID,
			getUserDataFromMessage(message.Message.From),
			// message.Message.ReplyToMessage.ForumTopicReopened.Name,
		)
	}

	if message.Message.ForumTopicEdited != nil {
		updatedName := ""

		if message.Message.ReplyToMessage != nil && message.Message.ReplyToMessage.ForumTopicCreated != nil {
			updatedName = message.Message.ReplyToMessage.ForumTopicCreated.Name
		}

		if message.Message.ForumTopicEdited != nil && updatedName != "" && message.Message.ForumTopicEdited.Name != updatedName {
			updatedName = fmt.Sprintf("%s (was %s)", message.Message.ForumTopicEdited.Name, updatedName)
		} else if message.Message.ForumTopicEdited != nil && updatedName == "" && message.Message.ForumTopicEdited.Name != updatedName {
			updatedName = fmt.Sprintf("General updated to %s", message.Message.ForumTopicEdited.Name)
		}

		return fmt.Sprintf(
			"M from %d (%s): Forum topic %s edited",
			message.Message.From.ID,
			getUserDataFromMessage(message.Message.From),
			updatedName,
		)
	}

	if message.Message.ForumTopicClosed != nil {
		return fmt.Sprintf(
			"M from %d (%s): Forum topic closed",
			message.Message.From.ID,
			getUserDataFromMessage(message.Message.From),
			// message.Message.ReplyToMessage.ForumTopicClosed.Name,
		)
	}

	if message.Message.ForwardOrigin != nil {
		return fmt.Sprintf(
			"F from %d (%s): %s",
			message.Message.From.ID,
			getUserDataFromMessage(message.Message.From),
			message.Message.Text,
		)
	}

	if message.Message.LeftChatMember != nil {
		return fmt.Sprintf(
			"LCM %d from %d (%s)",
			message.Message.LeftChatMember.ID,
			message.Message.Chat.ID,
			getUserDataFromMessage(message.Message.LeftChatMember),
		)
	}

	if message.Message.NewChatMembers != nil {
		return fmt.Sprintf(
			"NCM %d from %d (%s)",
			message.Message.NewChatMembers[0].ID,
			message.Message.Chat.ID,
			getUserDataFromMessage(&message.Message.NewChatMembers[0]),
		)
	}

	return fmt.Sprintf(
		"M from %d (%s): %s",
		message.Message.From.ID,
		getUserDataFromMessage(message.Message.From),
		message.Message.Text,
	)
}

func formatMEForLog(message *bm.Update) string {
	if message.EditedMessage.Chat.Type == "private" {
		if message.EditedMessage.Caption != "" {
			return fmt.Sprintf(
				"PME from %d (%s): Photo with message: %s",
				message.EditedMessage.From.ID,
				getUserDataFromMessage(message.EditedMessage.From),
				message.EditedMessage.Caption,
			)
		}

		return fmt.Sprintf("PME from %d (%s): %s", message.EditedMessage.Chat.ID, getChatDataFromMessage(&message.EditedMessage.Chat), message.EditedMessage.Text)
	}

	if message.EditedMessage.Caption != "" {
		return fmt.Sprintf(
			"ME from %d (%s): Photo with message: %s",
			message.EditedMessage.From.ID,
			getUserDataFromMessage(message.EditedMessage.From),
			message.EditedMessage.Caption,
		)
	}

	return fmt.Sprintf(
		"ME from %d (%s): %s",
		message.EditedMessage.From.ID,
		getUserDataFromMessage(message.EditedMessage.From),
		message.EditedMessage.Text,
	)
}

func formatChannelpostForLog(message *bm.Update) string {
	if message.ChannelPost.Chat.Type == "private" {
		return fmt.Sprintf("CP from %d (%s): %s", message.ChannelPost.Chat.ID, getChatDataFromMessage(&message.ChannelPost.Chat), message.ChannelPost.Text)
	}

	return fmt.Sprintf(
		"CP from %d (%s): %s",
		message.ChannelPost.From.ID,
		getUserDataFromMessage(message.ChannelPost.From),
		message.ChannelPost.Text,
	)
}

func formatMessageReactionForLog(message *bm.Update) string {
	oldEmoji := []string{}
	newEmoji := []string{}
	for _, reaction := range message.MessageReaction.OldReaction {
		oldEmoji = append(oldEmoji, reaction.ReactionTypeEmoji.Emoji)
	}
	for _, reaction := range message.MessageReaction.NewReaction {
		newEmoji = append(newEmoji, reaction.ReactionTypeEmoji.Emoji)
	}

	if message.MessageReaction.Chat.Type == "private" {
		return fmt.Sprintf("MR %s -> %s, by %d (%s)", strings.Join(oldEmoji, ","), strings.Join(newEmoji, ","), message.MessageReaction.User.ID, getChatDataFromMessage(&message.MessageReaction.Chat))
	}

	return fmt.Sprintf("MR %s -> %s, by %d (%s) in (https://t.me/%s/%d)", strings.Join(oldEmoji, ","), strings.Join(newEmoji, ","), message.MessageReaction.User.ID, getUserDataFromMessage(message.MessageReaction.User), message.MessageReaction.Chat.Username, message.MessageReaction.MessageID)
}

func formatCallbackQueryForLog(message *bm.Update) string {
	return fmt.Sprintf("CQ %s", message.CallbackQuery.Data)
}

func formatMessageReactionCountForLog(message *bm.Update) string {
	return fmt.Sprintf("MRC %#v", message.MessageReactionCount)
}

func formatChatJoinRequestForLog(message *bm.Update) string {
	return fmt.Sprintf(
		"CJR from %d (%s) -> %d",
		message.ChatJoinRequest.From.ID,
		getUserDataFromMessage(&message.ChatJoinRequest.From),
		message.ChatJoinRequest.Chat.ID,
	)
}

func getChatDataFromMessage(user *bm.Chat) string {
	if user == nil {
		return "Unknown"
	}

	fields := []string{}
	for _, field := range []string{user.FirstName, user.LastName, user.Username} {
		if field != "" {
			fields = append(fields, field)
		}
	}

	if len(fields) == 0 {
		return "Unknown"
	}

	return strings.Join(fields, " ")
}

func getUserDataFromMessage(user *bm.User) string {
	if user == nil {
		return "Unknown"
	}

	fields := []string{}
	for _, field := range []string{user.FirstName, user.LastName, user.Username} {
		if field != "" {
			fields = append(fields, field)
		}
	}

	if len(fields) == 0 {
		return "Unknown"
	}
	return strings.Join(fields, " ")
}
