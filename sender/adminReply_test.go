package sender

import (
	"errors"
	"net/url"
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestExtractTargetUserIDFromAdminReply(t *testing.T) {
	tests := []struct {
		name    string
		message *models.Message
		wantID  int64
		wantOK  bool
	}{
		{
			name: "notification message with ID",
			message: &models.Message{
				Text: "📝 Новая заявка на вступление\n\nID: 12345\nИмя: Test",
			},
			wantID: 12345,
			wantOK: true,
		},
		{
			name: "forwarded user message",
			message: &models.Message{
				ForwardOrigin: &models.MessageOrigin{
					Type: models.MessageOriginTypeUser,
					MessageOriginUser: &models.MessageOriginUser{
						SenderUser: models.User{ID: 777},
					},
				},
			},
			wantID: 777,
			wantOK: true,
		},
		{
			name: "not related message",
			message: &models.Message{
				Text: "hello",
			},
			wantID: 0,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := extractTargetUserIDFromAdminReply(tt.message)
			if gotID != tt.wantID || gotOK != tt.wantOK {
				t.Fatalf("extractTargetUserIDFromAdminReply() = (%d, %t), want (%d, %t)", gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestParseTelegramMessageTargetFromLink(t *testing.T) {
	tests := []struct {
		name            string
		link            string
		resolveUsername func(string) (int64, error)
		wantTarget      groupReplyTarget
		wantErr         error
	}{
		{
			name: "private supergroup link",
			link: "https://t.me/c/1234567890/321",
			wantTarget: groupReplyTarget{
				chatID:    -1001234567890,
				messageID: 321,
			},
		},
		{
			name: "forum link with thread in path",
			link: "https://t.me/c/1234567890/55/321",
			wantTarget: groupReplyTarget{
				chatID:          -1001234567890,
				messageID:       321,
				messageThreadID: 55,
			},
		},
		{
			name: "forum general topic link",
			link: "https://t.me/c/1234567890/1/541",
			wantTarget: groupReplyTarget{
				chatID:    -1001234567890,
				messageID: 541,
			},
		},
		{
			name: "public group link",
			link: "https://t.me/example_group/777",
			resolveUsername: func(username string) (int64, error) {
				if username != "example_group" {
					t.Fatalf("unexpected username: %s", username)
				}
				return -1009988776655, nil
			},
			wantTarget: groupReplyTarget{
				chatID:    -1009988776655,
				messageID: 777,
			},
		},
		{
			name: "tg privatepost link",
			link: "tg://privatepost?channel=-1008877665544&post=25&thread=8",
			wantTarget: groupReplyTarget{
				chatID:          -1008877665544,
				messageID:       25,
				messageThreadID: 8,
			},
		},
		{
			name:    "invalid telegram link",
			link:    "https://t.me/example_group/not-a-message-id",
			wantErr: errInvalidTelegramLink,
		},
		{
			name:    "not telegram link",
			link:    "https://example.com/path",
			wantErr: errNotTelegramMessageLink,
		},
		{
			name: "username resolve error",
			link: "https://t.me/example_group/100",
			resolveUsername: func(username string) (int64, error) {
				return 0, errors.New("chat not found")
			},
			wantErr: errors.New("chat not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := tt.resolveUsername
			if resolver == nil {
				resolver = func(string) (int64, error) {
					return 0, errors.New("resolver was called unexpectedly")
				}
			}

			got, err := parseTelegramMessageTargetFromLink(tt.link, resolver)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}

				if err.Error() != tt.wantErr.Error() {
					t.Fatalf("unexpected error: got %v, want %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.wantTarget {
				t.Fatalf("unexpected target: got %+v, want %+v", got, tt.wantTarget)
			}
		})
	}
}

func TestExtractThreadID(t *testing.T) {
	tests := []struct {
		name   string
		parts  []string
		query  string
		wantID int
	}{
		{
			name:   "thread from path",
			parts:  []string{"c", "123", "77", "500"},
			query:  "",
			wantID: 77,
		},
		{
			name:   "thread from query",
			parts:  []string{"c", "123", "500"},
			query:  "thread=44",
			wantID: 44,
		},
		{
			name:   "general thread from path",
			parts:  []string{"c", "123", "1", "500"},
			query:  "",
			wantID: 0,
		},
		{
			name:   "general thread from query",
			parts:  []string{"c", "123", "500"},
			query:  "thread=1",
			wantID: 0,
		},
		{
			name:   "topic from query",
			parts:  []string{"c", "123", "500"},
			query:  "topic=12",
			wantID: 12,
		},
		{
			name:   "no thread id",
			parts:  []string{"c", "123", "500"},
			query:  "single",
			wantID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("ParseQuery error: %v", err)
			}

			got := extractThreadID(tt.parts, values)
			if got != tt.wantID {
				t.Fatalf("extractThreadID() = %d, want %d", got, tt.wantID)
			}
		})
	}
}
