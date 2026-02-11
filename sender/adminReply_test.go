package sender

import (
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
