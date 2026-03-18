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
| `/queue list [page]` | Show queue with sections (Played/Now Playing/Up Next) |
| `/queue remove <position>` | Remove track from queue (1-indexed position, with autocomplete) |
| `/queue clear` | Clear queue (keeps current track) |
| `/queue restart` | Restart queue from the beginning |
| `/queue shuffle` | Shuffle the queue (current track stays at front) |
| `/queue seek <position>` | Jump to a specific position in the queue (1-indexed, with autocomplete) |
| `/loop [mode]` | Set loop mode (none/track/queue) or cycle through modes |
| `/autoplay <enabled>` | Enable or disable auto-play (automatically play related tracks when queue ends) |

## Architecture

### Layered Architecture

The module follows a hexagonal (ports & adapters) architecture with four layers:

- **Domain** — Pure business logic with no external dependencies. Contains entities (`Track`,
  `TrackList`, `Queue`, `PlayerState`, `User`), value objects (`QueueEntry`, `LoopMode`,
  `TrackSource`), domain services (`PlayerService`, `AutoPlayService`), domain events, repository
  interfaces, and the `TrackRecommender` interface.
- **Application** — Orchestrates domain operations through use cases and handles domain events.
  Defines port interfaces for infrastructure and exposes DTOs for the presentation layer. Each use
  case is a standalone struct in its own file.
- **Infrastructure** — Implements port interfaces and domain repository interfaces with concrete
  technology (Lavalink, Discord API, in-memory stores, YouTube API).
- **Presentation** — Maps Discord slash commands, autocomplete requests, and gateway events to use
  case calls. Includes `EventHandlers` for reacting to Discord gateway events (e.g., auto-leave when
  the bot is disconnected from a voice channel).
- **Platforms** — Thin type adapters that bridge generic port interfaces to platform-specific types
  (e.g., Discord snowflake IDs for voice connections and now-playing destinations).

### Platform-Agnostic Domain

The domain layer has no dependency on Discord or any other platform. Platform-specific concepts are
abstracted:

- `PlayerStateID` (UUID v7) replaces guild snowflake IDs as the aggregate identity
- `UserID` (string) replaces Discord user snowflake IDs
- `TrackSource` (enum) replaces Lavalink source names
- Port interfaces in `application/ports/` use generics (`VoiceConnectionGateway[T]`,
  `UserVoiceStateProvider[C, P]`, `NowPlayingDestinationSetter[T]`) so the application layer stays
  platform-neutral while the `platforms/` layer supplies Discord-specific type parameters

#### PlayerService (Domain Service)

`PlayerService` wraps `PlayerState` mutations and produces the correct domain events for each
operation. It also integrates `AutoPlayService` for automatic track recommendations when the queue
is exhausted.

Key responsibilities:

- Append tracks → `TrackAddedEvent` (+ `TrackStartedEvent` if playback begins)
- Prepend/Insert tracks → `TrackAddedEvent`
- Seek/Skip → `TrackStartedEvent` or `QueueExhaustedEvent`
- Remove → `TrackRemovedEvent` (+ `TrackStartedEvent` or `PlaybackStoppedEvent`)
- Clear → `QueueClearedEvent` (+ `PlaybackStoppedEvent`)
- ClearExceptCurrent → `QueueClearedEvent`
- Pause/Resume → `PlaybackPausedEvent` / `PlaybackResumedEvent`
- Shuffle → `QueueShuffledEvent`
- TryAutoPlay → `TrackAddedEvent` + `TrackStartedEvent` (on success)

#### AutoPlayService (Domain Service)

`AutoPlayService` recommends the next track when the queue is exhausted. It selects seeds from the
current queue and delegates to a `TrackRecommender` for recommendations.

Seed selection strategy:

- Up to 2 randomly sampled manually added tracks
- Up to 1 most recent auto-play track
- Seeds are filtered through `TrackRecommender.AcceptsSeed()`
- All non-seed tracks are passed as exclusions

`PlayerService.TryAutoPlay` calls `AutoPlayService` when auto-play is enabled on the player state.
If a recommendation is found, it is appended and playback begins immediately.

### Index-Based Queue Model

The music player uses an index-based queue instead of a traditional pop-based queue. This design
enables loop functionality while maintaining track history:

- **currentIndex**: Tracks position in the queue (-1 before start, 0+ during playback)
- **Tracks are never removed** when finished—the index advances instead
- **Loop modes** determine advancement behavior when a track ends

| Loop Mode | Behavior |
| --------- | -------- |
| `none` | Advance to next track; stop when queue ends |
| `track` | Repeat current track indefinitely |
| `queue` | Advance with wrap-around (restart from beginning after last track) |

Queue sections displayed in `/queue list`:

- **Played**: Tracks before currentIndex, or all queued tracks when playback is inactive
- **Now Playing**: Track at currentIndex
- **Up Next**: Tracks after currentIndex

### Event-Driven Design

The music player uses a channel-based event bus for async, decoupled operations. This ensures
Discord interaction responses are sent immediately while background tasks (playback, notifications)
happen asynchronously.

Events are defined as domain events (`domain/events.go`) implementing a sealed `Event` interface.
The `EventPublisher` port publishes one or more events, and the `EventSubscriber` port allows
subscribing to specific event types by `reflect.Type`.

### Domain Events (`domain/events.go`)

| Event | Description |
| ----- | ----------- |
| `TrackAddedEvent` | One or more tracks added to the queue |
| `TrackRemovedEvent` | A track removed from the queue |
| `TrackStartedEvent` | A track started playing (new current track) |
| `TrackEndedEvent` | A track finished or failed to load (from audio gateway) |
| `PlaybackStartedEvent` | Playback became active |
| `PlaybackStoppedEvent` | Playback became inactive (queue ended, cleared, last track removed) |
| `PlaybackPausedEvent` | Playback paused |
| `PlaybackResumedEvent` | Playback resumed |
| `QueueClearedEvent` | Queue cleared (carries count) |
| `QueueExhaustedEvent` | Queue ran out of tracks (after skip/remove with no next track) |
| `QueueShuffledEvent` | Queue order randomized |

### Domain Event Handlers (`application/domain_event_handlers.go`)

`DomainEventHandlers` coordinates application-level side effects:

- **HandleTrackStarted**: Updates the now-playing display
- **HandleTrackEnded**: Advances the queue based on loop mode, removes failed tracks, attempts
  auto-play when exhausted, triggers next track playback
- **HandlePlaybackStopped**: Clears the now-playing display
- **HandleQueueExhausted**: Attempts auto-play; if no recommendation, stops playback

### Infrastructure Adapters

**Discord / Lavalink** (`infrastructure/discord/lavalink/`):

- `AudioGateway` — playback control via Lavalink
- `TrackRepository` — track loading via Lavalink (with in-memory cache)
- `TrackResolver` — track search via Lavalink

**Discord** (`infrastructure/discord/`):

- `VoiceConnectionGateway` — voice channel join/leave via discordgo + Lavalink, with bidirectional
  guild-to-player-state mapping. Also implements `PlayerStateLocator`.
- `NowPlayingGateway` — sends Discord embeds with source-specific colors, icons, and artwork
- `UserRepository` — fetches user display info from Discord
- `UserVoiceStateProvider` — queries Discord session for user voice state

**In-Memory** (`infrastructure/in_memory/`):

- `EventBus` — channel-based event bus implementing both `EventPublisher` and `EventSubscriber`
- `PlayerStateRepository` — in-memory `PlayerStateRepository`

**YouTube** (`infrastructure/youtube/`):

- `TrackRecommender` — YouTube mix-based track recommendations for auto-play
