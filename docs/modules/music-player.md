# `music_player` Module

## Tech Stack

| Component | Technology |
| --------- | ---------- |
| Lavalink Client | `github.com/disgoorg/disgolink/v3` |

## Configuration

Environment variables:

- `LAVALINK_ADDRESS`: (required)
- `LAVALINK_PASSWORD`: (required)

## Commands

| Command | Description |
| ------- | ----------- |
| `/join [channel]` | Join user's voice channel or channel specified |
| `/leave` | Leave voice channel |
| `/play <query>` | Play track (URL or search, with autocomplete) |
| `/stop` | Stop playback |
| `/pause` | Pause playback |
| `/resume` | Resume playback |
| `/skip` | Skip to next track |
| `/queue list [page]` | Show queue (paginated) |
| `/queue remove <position>` | Remove track from queue (0 = current track, with autocomplete) |
| `/queue clear` | Clear queue (keeps current track) |

## Project Structure

```text
internal/modules/music_player/
├── module.go                    # Module entry point (bot.Module implementation)
├── config.go                    # Environment configuration loading
├── domain/
│   ├── track.go                 # Track entity
│   ├── queue.go                 # Queue entity
│   ├── player_state.go          # PlayerState aggregate root
│   ├── search_query.go          # SearchQuery value object
│   ├── source.go                # TrackSource value object
│   └── repository.go            # PlayerStateRepository interface
├── application/
│   ├── ports/
│   │   ├── types.go             # Shared types (LoadResult, TrackInfo, etc.)
│   │   ├── audio_player.go      # AudioPlayer interface
│   │   ├── voice_connection.go  # VoiceConnection interface
│   │   ├── voice_state.go       # VoiceStateProvider interface
│   │   ├── track_resolver.go    # TrackResolver interface
│   │   ├── notification.go      # NotificationSender interface
│   │   └── event_publisher.go   # EventPublisher interface and event types
│   ├── usecases/
│   │   ├── errors.go            # Use case error definitions
│   │   ├── playback.go          # PlaybackService (pause, resume, skip, play next)
│   │   ├── queue.go             # QueueService (add, list, remove, clear)
│   │   ├── track_loader.go      # TrackLoaderService (load, search tracks)
│   │   ├── voice_channel.go     # VoiceChannelService (join, leave)
│   │   ├── autocomplete.go      # AutocompleteService (queue tracks, search)
│   │   └── types.go             # Type re-exports for presentation layer
│   └── events/
│       ├── types.go             # Re-exports event types from ports
│       ├── bus.go               # Event bus implementation
│       └── handlers.go          # PlaybackEventHandler, NotificationEventHandler
├── infrastructure/
│   ├── lavalink_client.go       # LavalinkAdapter (implements AudioPlayer, VoiceConnection, TrackResolver)
│   ├── memory_repository.go     # In-memory PlayerStateRepository
│   ├── voice_state.go           # VoiceStateAdapter (implements VoiceStateProvider)
│   └── notifier.go              # DiscordNotifier (implements NotificationSender)
└── presentation/
    ├── commands.go              # Slash command definitions
    ├── handlers.go              # Command interaction handlers
    ├── autocomplete.go          # Autocomplete for /play and /queue remove
    └── event_handlers.go        # Discord gateway event handlers (VoiceStateUpdate)
```

## Architecture

### Event-Driven Design

The music player uses a channel-based event bus for async, decoupled operations.
This ensures Discord interaction responses are sent immediately while background
tasks (playback, notifications) happen asynchronously.

```text
/play command
  │
  ├─► QueueService.Add()
  │     └─► publish TrackEnqueuedEvent ──────┐
  │                                          │
  └─► respond "Added to Queue" ◄─────────────┼─── (immediate response)
                                             │
         EventBus (async goroutines)         │
              │                              │
              ├─► PlaybackEventHandler ◄─────┘
              │     └─► PlaybackService.PlayNext()
              │           └─► publish PlaybackStartedEvent ───┐
              │                                               │
              └─► NotificationEventHandler ◄──────────────────┘
                    └─► send "Now Playing" message (async)
```

### Event Types (`application/ports/event_publisher.go`)

| Event | Published By | Consumed By | Description |
| ----- | ------------ | ----------- | ----------- |
| `TrackEnqueuedEvent` | QueueService | PlaybackEventHandler | Track added to queue |
| `PlaybackStartedEvent` | PlaybackService | NotificationEventHandler | Track started playing |
| `PlaybackFinishedEvent` | PlaybackEventHandler, PlaybackService, VoiceChannelService | NotificationEventHandler | Track finished, delete "Now Playing" message |
| `TrackEndedEvent` | LavalinkAdapter | PlaybackEventHandler | Track finished (from Lavalink) |

### Track End Reasons (`application/ports/event_publisher.go`)

When a track ends, Lavalink provides a reason. Only certain reasons advance the
queue:

| Reason | Advances Queue | Description |
| ------ | -------------- | ----------- |
| `finished` | Yes | Track completed normally |
| `load_failed` | Yes | Track failed to load |
| `stopped` | No | User stopped playback |
| `replaced` | No | Track was replaced by another |
| `cleanup` | No | Player was cleaned up |

### Event Handlers (`application/events/handlers.go`)

- **PlaybackEventHandler**: Listens for `TrackEnqueuedEvent` (auto-start if
  idle) and `TrackEndedEvent` (play next track if reason allows)
- **NotificationEventHandler**: Listens for `PlaybackStartedEvent` (send "Now
  Playing") and `PlaybackFinishedEvent` (delete message)

### Port Interfaces (`application/ports/`)

```go
// audio_player.go
type AudioPlayer interface {
    Play(ctx context.Context, guildID snowflake.ID, track *domain.Track) error
    Stop(ctx context.Context, guildID snowflake.ID) error
    Pause(ctx context.Context, guildID snowflake.ID) error
    Resume(ctx context.Context, guildID snowflake.ID) error
}

// voice_connection.go
type VoiceConnection interface {
    JoinChannel(ctx context.Context, guildID, channelID snowflake.ID) error
    LeaveChannel(ctx context.Context, guildID snowflake.ID) error
}

// track_resolver.go
type TrackResolver interface {
    LoadTracks(ctx context.Context, query string) (*LoadResult, error)
}

// voice_state.go
type VoiceStateProvider interface {
    GetUserVoiceChannel(guildID, userID snowflake.ID) (snowflake.ID, error)
}

// notification.go
type NotificationSender interface {
    SendNowPlaying(channelID snowflake.ID, info *NowPlayingInfo) (snowflake.ID, error)
    DeleteMessage(channelID, messageID snowflake.ID) error
    SendQueueAdded(channelID snowflake.ID, info *QueueAddedInfo) error
    SendError(channelID snowflake.ID, message string) error
}

// event_publisher.go
type EventPublisher interface {
    PublishTrackEnqueued(event TrackEnqueuedEvent)
    PublishPlaybackStarted(event PlaybackStartedEvent)
    PublishPlaybackFinished(event PlaybackFinishedEvent)
    PublishTrackEnded(event TrackEndedEvent)
}
```

### Infrastructure Adapters

The `LavalinkAdapter` (`infrastructure/lavalink_client.go`) implements three
port interfaces:

- `AudioPlayer` - playback control via Lavalink
- `VoiceConnection` - voice channel join/leave via discordgo + Lavalink
- `TrackResolver` - track loading/searching via Lavalink

Voice connection uses a two-phase handshake, waiting for both
`VoiceStateUpdate` and `VoiceServerUpdate` events before completing.
