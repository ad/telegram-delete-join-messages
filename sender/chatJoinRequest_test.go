package sender

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestExtractUserIDFromAdminNotification(t *testing.T) {
	tests := []struct {
		name    string
		message *models.Message
		wantID  int64
		wantOK  bool
	}{
		{
			name: "join request notification",
			message: &models.Message{
				Text: "📝 Новая заявка на вступление\n\nID: 12345\nИмя: Test",
			},
			wantID: 12345,
			wantOK: true,
		},
		{
			name: "joined notification",
			message: &models.Message{
				Text: "✅ Пользователь присоединился к группе\n\nID: 42\nUsername: @user",
			},
			wantID: 42,
			wantOK: true,
		},
		{
			name: "private message relay notification",
			message: &models.Message{
				Text: "📨 ЛС от пользователя\n\nID: 71557\nИмя: Test",
			},
			wantID: 71557,
			wantOK: true,
		},
		{
			name: "non notification text",
			message: &models.Message{
				Text: "ID: 7",
			},
			wantID: 0,
			wantOK: false,
		},
		{
			name: "notification without id",
			message: &models.Message{
				Text: "👋 Пользователь вышел из группы\n\nИмя: User",
			},
			wantID: 0,
			wantOK: false,
		},
		{
			name:   "nil message",
			wantID: 0,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := extractUserIDFromAdminNotification(tt.message)
			if gotID != tt.wantID || gotOK != tt.wantOK {
				t.Fatalf("extractUserIDFromAdminNotification() = (%d, %t), want (%d, %t)", gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}
