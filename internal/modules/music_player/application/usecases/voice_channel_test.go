package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/events"
)

func TestVoiceChannelService_Join(t *testing.T) {
	guildID := snowflake.ID(1)
	userID := snowflake.ID(2)
	notificationChannelID := snowflake.ID(3)
	voiceChannelID := snowflake.ID(4)

	tests := []struct {
		name               string
		input              JoinInput
		setupRepo          func(*mockRepository)
		setupConnection    func(*mockVoiceConnection)
		setupVoice         func(*mockVoiceStateProvider)
		wantErr            error
		wantVoiceChannelID snowflake.ID
	}{
		{
			name: "join user's channel",
			input: JoinInput{
				GuildID:               guildID,
				UserID:                userID,
				NotificationChannelID: notificationChannelID,
			},
			setupVoice: func(m *mockVoiceStateProvider) {
				m.channels[userID] = voiceChannelID
			},
			wantVoiceChannelID: voiceChannelID,
		},
		{
			name: "join specific channel",
			input: JoinInput{
				GuildID:               guildID,
				UserID:                userID,
				NotificationChannelID: notificationChannelID,
				VoiceChannelID:        voiceChannelID,
			},
			wantVoiceChannelID: voiceChannelID,
		},
		{
			name: "user not in voice",
			input: JoinInput{
				GuildID:               guildID,
				UserID:                userID,
				NotificationChannelID: notificationChannelID,
			},
			setupVoice: func(m *mockVoiceStateProvider) {
				// No channel for user
			},
			wantErr: ErrUserNotInVoice,
		},
		{
			name: "already connected to same channel updates notification channel",
			input: JoinInput{
				GuildID:               guildID,
				UserID:                userID,
				NotificationChannelID: snowflake.ID(99), // Different notification channel
				VoiceChannelID:        voiceChannelID,
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
			},
			wantVoiceChannelID: voiceChannelID,
		},
		{
			name: "voice connection error",
			input: JoinInput{
				GuildID:               guildID,
				UserID:                userID,
				NotificationChannelID: notificationChannelID,
				VoiceChannelID:        voiceChannelID,
			},
			setupConnection: func(m *mockVoiceConnection) {
				m.joinErr = errors.New("connection failed")
			},
			wantErr: errors.New("connection failed"),
		},
		{
			name: "move to different channel preserves queue",
			input: JoinInput{
				GuildID:               guildID,
				UserID:                userID,
				NotificationChannelID: notificationChannelID,
				VoiceChannelID:        snowflake.ID(999), // Different voice channel
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				// Add tracks to queue
				state.Queue.Add(mockTrack("track-1"))
				state.Queue.Add(mockTrack("track-2"))
			},
			wantVoiceChannelID: snowflake.ID(999),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			connection := &mockVoiceConnection{}
			voiceState := &mockVoiceStateProvider{channels: make(map[snowflake.ID]snowflake.ID)}

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			if tt.setupConnection != nil {
				tt.setupConnection(connection)
			}
			if tt.setupVoice != nil {
				tt.setupVoice(voiceState)
			}

			service := NewVoiceChannelService(repo, connection, voiceState, nil)
			output, err := service.Join(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if output.VoiceChannelID != tt.wantVoiceChannelID {
				t.Errorf(
					"expected channel ID %d, got %d",
					tt.wantVoiceChannelID,
					output.VoiceChannelID,
				)
			}

			// Verify state was created
			state := repo.Get(guildID)
			if state == nil {
				t.Fatal("expected state to exist")
			}
			if state.VoiceChannelID != tt.wantVoiceChannelID {
				t.Errorf(
					"expected state channel ID %d, got %d",
					tt.wantVoiceChannelID,
					state.VoiceChannelID,
				)
			}
		})
	}
}

func TestVoiceChannelService_Join_PreservesQueueOnMove(t *testing.T) {
	guildID := snowflake.ID(1)
	userID := snowflake.ID(2)
	notificationChannelID := snowflake.ID(3)
	oldVoiceChannel := snowflake.ID(4)
	newVoiceChannel := snowflake.ID(999)

	repo := newMockRepository()
	connection := &mockVoiceConnection{}

	// Create existing state with tracks in queue
	state := repo.createConnectedState(guildID, oldVoiceChannel, notificationChannelID)
	state.Queue.Add(mockTrack("track-1"))
	state.Queue.Add(mockTrack("track-2"))

	service := NewVoiceChannelService(repo, connection, nil, nil)

	// Move to different channel
	output, err := service.Join(context.Background(), JoinInput{
		GuildID:               guildID,
		UserID:                userID,
		NotificationChannelID: notificationChannelID,
		VoiceChannelID:        newVoiceChannel,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.VoiceChannelID != newVoiceChannel {
		t.Errorf("expected channel ID %d, got %d", newVoiceChannel, output.VoiceChannelID)
	}

	// Verify state was updated, not recreated
	updatedState := repo.Get(guildID)
	if updatedState == nil {
		t.Fatal("expected state to exist")
	}

	// Verify voice channel was updated
	if updatedState.VoiceChannelID != newVoiceChannel {
		t.Errorf(
			"expected state channel ID %d, got %d",
			newVoiceChannel,
			updatedState.VoiceChannelID,
		)
	}

	// Verify queue was preserved
	if updatedState.Queue.Len() != 2 {
		t.Errorf("expected queue length 2, got %d", updatedState.Queue.Len())
	}

	tracks := updatedState.Queue.List()
	if tracks[0].ID != "track-1" {
		t.Errorf("expected first track ID 'track-1', got %q", tracks[0].ID)
	}
	if tracks[1].ID != "track-2" {
		t.Errorf("expected second track ID 'track-2', got %q", tracks[1].ID)
	}
}

func TestVoiceChannelService_Leave(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	notificationChannelID := snowflake.ID(3)

	tests := []struct {
		name            string
		input           LeaveInput
		setupRepo       func(*mockRepository)
		setupConnection func(*mockVoiceConnection)
		wantErr         error
	}{
		{
			name: "leave successfully",
			input: LeaveInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
			},
		},
		{
			name: "not connected",
			input: LeaveInput{
				GuildID: guildID,
			},
			wantErr: ErrNotConnected,
		},
		{
			name: "voice connection error",
			input: LeaveInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
			},
			setupConnection: func(m *mockVoiceConnection) {
				m.leaveErr = errors.New("disconnect failed")
			},
			wantErr: errors.New("disconnect failed"),
		},
		{
			name: "deletes state on leave",
			input: LeaveInput{
				GuildID: guildID,
			},
			setupRepo: func(m *mockRepository) {
				state := m.createConnectedState(guildID, voiceChannelID, notificationChannelID)
				state.Queue.Add(mockTrack("track-1"))
				state.Queue.Add(mockTrack("track-2"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			connection := &mockVoiceConnection{}

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			if tt.setupConnection != nil {
				tt.setupConnection(connection)
			}

			service := NewVoiceChannelService(repo, connection, nil, nil)
			err := service.Leave(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify state was deleted
			if len(repo.deleted) == 0 {
				t.Error("expected state to be deleted")
			}
			if repo.deleted[0] != guildID {
				t.Errorf("expected deleted guild ID %d, got %d", guildID, repo.deleted[0])
			}
		})
	}
}

func TestVoiceChannelService_Leave_PublishesPlaybackFinishedEvent(t *testing.T) {
	guildID := snowflake.ID(1)
	notificationChannelID := snowflake.ID(3)
	voiceChannelID := snowflake.ID(4)
	nowPlayingMsgID := snowflake.ID(999)

	repo := newMockRepository()
	connection := &mockVoiceConnection{}
	bus := events.NewBus(10)
	defer bus.Close()

	// Create connected state with a now playing message
	state := repo.createConnectedState(guildID, voiceChannelID, notificationChannelID)
	state.SetNowPlayingMessageID(nowPlayingMsgID)

	service := NewVoiceChannelService(repo, connection, nil, bus)

	// Start listening for the event
	eventReceived := make(chan events.PlaybackFinishedEvent, 1)
	go func() {
		select {
		case event := <-bus.PlaybackFinished():
			eventReceived <- event
		case <-time.After(time.Second):
		}
	}()

	err := service.Leave(context.Background(), LeaveInput{GuildID: guildID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the event was published
	select {
	case event := <-eventReceived:
		if event.GuildID != guildID {
			t.Errorf("expected GuildID %d, got %d", guildID, event.GuildID)
		}
		if event.NotificationChannelID != notificationChannelID {
			t.Errorf(
				"expected NotificationChannelID %d, got %d",
				notificationChannelID,
				event.NotificationChannelID,
			)
		}
		if event.LastMessageID == nil || *event.LastMessageID != nowPlayingMsgID {
			t.Errorf("expected LastMessageID %d, got %v", nowPlayingMsgID, event.LastMessageID)
		}
	case <-time.After(time.Second):
		t.Error("expected PlaybackFinishedEvent to be published")
	}
}
