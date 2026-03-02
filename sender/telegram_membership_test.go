package sender

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestDidLeaveGroup(t *testing.T) {
	user := &models.User{ID: 42}

	tests := []struct {
		name string
		old  models.ChatMember
		new  models.ChatMember
		want bool
	}{
		{
			name: "member to left",
			old: models.ChatMember{
				Type:   models.ChatMemberTypeMember,
				Member: &models.ChatMemberMember{User: user},
			},
			new: models.ChatMember{
				Type: models.ChatMemberTypeLeft,
				Left: &models.ChatMemberLeft{User: user},
			},
			want: true,
		},
		{
			name: "restricted member to restricted not member",
			old: models.ChatMember{
				Type: models.ChatMemberTypeRestricted,
				Restricted: &models.ChatMemberRestricted{
					User:     user,
					IsMember: true,
				},
			},
			new: models.ChatMember{
				Type: models.ChatMemberTypeRestricted,
				Restricted: &models.ChatMemberRestricted{
					User:     user,
					IsMember: false,
				},
			},
			want: true,
		},
		{
			name: "restricted member unchanged",
			old: models.ChatMember{
				Type: models.ChatMemberTypeRestricted,
				Restricted: &models.ChatMemberRestricted{
					User:     user,
					IsMember: true,
				},
			},
			new: models.ChatMember{
				Type: models.ChatMemberTypeRestricted,
				Restricted: &models.ChatMemberRestricted{
					User:     user,
					IsMember: true,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := didLeaveGroup(tt.old, tt.new)
			if got != tt.want {
				t.Fatalf("didLeaveGroup() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestDidJoinGroup(t *testing.T) {
	user := &models.User{ID: 100}

	tests := []struct {
		name string
		old  models.ChatMember
		new  models.ChatMember
		want bool
	}{
		{
			name: "left to member",
			old: models.ChatMember{
				Type: models.ChatMemberTypeLeft,
				Left: &models.ChatMemberLeft{User: user},
			},
			new: models.ChatMember{
				Type:   models.ChatMemberTypeMember,
				Member: &models.ChatMemberMember{User: user},
			},
			want: true,
		},
		{
			name: "restricted not member to member",
			old: models.ChatMember{
				Type: models.ChatMemberTypeRestricted,
				Restricted: &models.ChatMemberRestricted{
					User:     user,
					IsMember: false,
				},
			},
			new: models.ChatMember{
				Type:   models.ChatMemberTypeMember,
				Member: &models.ChatMemberMember{User: user},
			},
			want: true,
		},
		{
			name: "member to restricted member",
			old: models.ChatMember{
				Type:   models.ChatMemberTypeMember,
				Member: &models.ChatMemberMember{User: user},
			},
			new: models.ChatMember{
				Type: models.ChatMemberTypeRestricted,
				Restricted: &models.ChatMemberRestricted{
					User:     user,
					IsMember: true,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := didJoinGroup(tt.old, tt.new)
			if got != tt.want {
				t.Fatalf("didJoinGroup() = %t, want %t", got, tt.want)
			}
		})
	}
}
