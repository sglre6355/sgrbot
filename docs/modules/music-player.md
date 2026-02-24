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
| `/loop [mode]` | Set loop mode (none/track/queue) or cycle through modes |
| `/queue list [page]` | Show queue with sections (Played/Now Playing/Up Next) |
| `/queue remove <position>` | Remove track from queue (1-indexed position, with autocomplete) |
| `/queue clear` | Clear queue (keeps current track) |
| `/queue restart` | Restart queue from the beginning |
| `/queue shuffle` | Shuffle the queue (current track stays at front) |
| `/queue seek <position>` | Jump to a specific position in the queue (1-indexed, with autocomplete) |

## Project Structure

```text
internal/modules/music_player/
├── module.go                    # Module entry point (bot.Module implementation)
├── config.go                    # Environment configuration loading
├── domain/
│   ├── track.go                 # Track entity and TrackID
│   ├── queue_entry.go           # QueueEntry value object
│   ├── queue.go                 # Queue entity and QueueRepository interface
│   ├── player_state.go          # PlayerState aggregate root and PlayerStateRepository interface
│   ├── loop_mode.go             # LoopMode value object (none/track/queue)
│   ├── events.go                # Domain events (TrackEndedEvent, CurrentTrackChangedEvent)
│   ├── source.go                # TrackSource value object
│   ├── now_playing.go           # NowPlayingMessage value object
│   └── track_list.go            # TrackListType and TrackList value objects
├── application/
│   ├── ports/
│   │   ├── audio_player.go      # AudioPlayer interface
│   │   ├── voice_connection.go  # VoiceConnection interface
│   │   ├── voice_state.go       # VoiceStateProvider interface
│   │   ├── track_provider.go    # TrackProvider interface
│   │   ├── notification.go      # NotificationSender interface
│   │   ├── event_publisher.go   # EventPublisher interface
│   │   ├── event_subscriber.go  # EventSubscriber interface
│   │   └── user_info.go         # UserInfoProvider interface and UserInfo DTO
│   ├── usecases/
│   │   ├── errors.go            # Use case error definitions
│   │   ├── playback.go          # PlaybackService (pause, resume, skip, loop)
│   │   ├── queue.go             # QueueService (add, list, remove, clear, shuffle, restart, seek)
│   │   ├── track_loader.go      # TrackLoaderService (load tracks, resolve queries)
│   │   ├── voice_channel.go     # VoiceChannelService (join, leave, handle bot voice state)
│   │   └── notification_channel.go # NotificationChannelService (set notification channel)
│   └── event_handlers.go        # PlaybackEventHandler, NotificationEventHandler
├── infrastructure/
│   ├── lavalink_client.go       # LavalinkAdapter (implements AudioPlayer, VoiceConnection, TrackProvider)
│   ├── memory_repository.go     # In-memory PlayerStateRepository
│   ├── channel_event_bus.go     # ChannelEventBus (implements EventPublisher, EventSubscriber)
│   ├── voice_state.go           # VoiceStateProvider (implements VoiceStateProvider)
│   ├── discord_user_info.go     # DiscordUserInfoProvider (implements UserInfoProvider)
│   └── notifier.go              # Notifier (implements NotificationSender)
└── presentation/
    └── discord/
        ├── commands.go           # Slash command definitions
        ├── command_handlers.go   # Command interaction handlers
        ├── autocomplete.go       # Autocomplete for /play, /queue remove, /queue seek
        └── event_handlers.go     # Discord gateway event handlers (VoiceStateUpdate)
```

## Architecture

### Index-Based Queue Model

The music player uses an index-based queue instead of a traditional pop-based
queue. This design enables loop functionality while maintaining track history:

- **currentIndex**: Tracks position in the queue (-1 before start, 0+ during playback)
- **Tracks are never removed** when finished—the index advances instead
- **Loop modes** determine advancement behavior when a track ends

| Loop Mode | Behavior |
| --------- | -------- |
| `none` | Advance to next track; stop when queue ends |
| `track` | Repeat current track indefinitely |
| `queue` | Advance with wrap-around (restart from beginning after last track) |

Queue sections displayed in `/queue list`:

- **Played**: Tracks before currentIndex (previously played)
- **Now Playing**: Track at currentIndex
- **Up Next**: Tracks after currentIndex

### Event-Driven Design

The music player uses a channel-based event bus for async, decoupled operations.
This ensures Discord interaction responses are sent immediately while background
tasks (playback, notifications) happen asynchronously.

Events are defined as domain events (`domain/events.go`) implementing a sealed
`Event` interface. The `EventPublisher` port has a single `Publish(event)` method,
and the `EventSubscriber` port allows subscribing to specific event types by
`reflect.Type`.

```text
/play command
  │
  ├─► QueueService.Add()
  │     └─► publish CurrentTrackChangedEvent ──────┐
  │                                                │
  └─► respond "Added to Queue" ◄───────────────────┼──(immediate response)
                                                   │
         EventBus (async goroutines)               │
              │                                    │
              ├─► PlaybackEventHandler ◄───────────┤
              │     └─► AudioPlayer.Play(trackID)  │
              │                                    │
              └─► NotificationEventHandler ◄───────┘
                    └─► send "Now Playing" embed (async)
```

### Domain Events (`domain/events.go`)

| Event | Published By | Consumed By | Description |
| ----- | ------------ | ----------- | ----------- |
| `CurrentTrackChangedEvent` | QueueService (add, seek, restart, clear, remove) | PlaybackEventHandler, NotificationEventHandler | Queue index changed; triggers playback and notification |
| `TrackEndedEvent` | LavalinkAdapter | PlaybackEventHandler | Track finished (from Lavalink); carries `ShouldAdvanceQueue` and `TrackFailed` flags |

The `TrackEndedEvent.ShouldAdvanceQueue` flag encapsulates Lavalink's track end
reasons. Only `finished` and `load_failed` reasons advance the queue; `stopped`,
`replaced`, and `cleanup` do not. When `TrackFailed` is true, the failed track
is removed from the queue.

### Event Handlers (`application/event_handlers.go`)

- **PlaybackEventHandler**: Subscribes to `CurrentTrackChangedEvent` (start
  playback of current track or stop if none) and `TrackEndedEvent` (advance
  queue based on loop mode, remove failed tracks, publish new
  `CurrentTrackChangedEvent`)
- **NotificationEventHandler**: Subscribes to `CurrentTrackChangedEvent` (delete
  previous "Now Playing" message, send new one if a track is current)

### Port Interfaces (`application/ports/`)

```go
// audio_player.go
type AudioPlayer interface {
    Play(ctx context.Context, guildID snowflake.ID, trackID domain.TrackID) error
    Stop(ctx context.Context, guildID snowflake.ID) error
    Pause(ctx context.Context, guildID snowflake.ID) error
    Resume(ctx context.Context, guildID snowflake.ID) error
}

// voice_connection.go
type VoiceConnection interface {
    JoinChannel(ctx context.Context, guildID, channelID snowflake.ID) error
    LeaveChannel(ctx context.Context, guildID snowflake.ID) error
}

// track_provider.go
type TrackProvider interface {
    LoadTrack(ctx context.Context, id domain.TrackID) (domain.Track, error)
    LoadTracks(ctx context.Context, ids ...domain.TrackID) ([]domain.Track, error)
    ResolveQuery(ctx context.Context, query string) (domain.TrackList, error)
}

// voice_state.go
type VoiceStateProvider interface {
    GetUserVoiceChannel(guildID, userID snowflake.ID) (*snowflake.ID, error)
}

// notification.go
type NotificationSender interface {
    SendNowPlaying(guildID, channelID snowflake.ID, trackID domain.TrackID,
        requesterID snowflake.ID, enqueuedAt time.Time) (snowflake.ID, error)
    DeleteMessage(channelID, messageID snowflake.ID) error
    SendError(channelID snowflake.ID, message string) error
}

// event_publisher.go
type EventPublisher interface {
    Publish(event domain.Event) error
}

// event_subscriber.go
type EventSubscriber interface {
    Subscribe(eventType reflect.Type, handler func(context.Context, domain.Event)) error
}

// user_info.go
type UserInfoProvider interface {
    GetUserInfo(guildID, userID snowflake.ID) (*UserInfo, error)
}
```

### Infrastructure Adapters

The `LavalinkAdapter` (`infrastructure/lavalink_client.go`) implements three
port interfaces:

- `AudioPlayer` - playback control via Lavalink (plays by TrackID using an
  internal encoded track cache)
- `VoiceConnection` - voice channel join/leave via discordgo + Lavalink
- `TrackProvider` - track loading/searching via Lavalink (caches both encoded
  track data and domain Track objects)

Voice connection uses a buffered two-phase handshake, handling out-of-order
`VoiceStateUpdate` and `VoiceServerUpdate` events before forwarding to Lavalink.

Other adapters:

- `ChannelEventBus` - channel-based event bus implementing both `EventPublisher`
  and `EventSubscriber`
- `MemoryRepository` - in-memory `PlayerStateRepository`
- `VoiceStateProvider` - queries Discord session for user voice state
- `DiscordUserInfoProvider` - fetches user display name and avatar from Discord
  (nick > globalName > username)
- `Notifier` - sends Discord embeds with source-specific colors, icons, and
  YouTube/Twitch thumbnail resolution
