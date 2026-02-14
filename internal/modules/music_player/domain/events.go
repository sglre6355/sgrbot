package domain

import (
	"github.com/disgoorg/snowflake/v2"
)

type Event interface {
	isEvent()
}

// TrackEndedEvent is published when track ends.
type TrackEndedEvent struct {
	GuildID            snowflake.ID
	ShouldAdvanceQueue bool
	TrackFailed        bool
}

func NewTrackEndedEvent(
	guildID snowflake.ID,
	shouldAdvanceQueue bool,
	trackFailed bool,
) TrackEndedEvent {
	return TrackEndedEvent{
		GuildID:            guildID,
		ShouldAdvanceQueue: shouldAdvanceQueue,
		TrackFailed:        trackFailed,
	}
}

func (e TrackEndedEvent) isEvent() {}

// CurrentTrackChangedEvent is published when queue index is modified.
type CurrentTrackChangedEvent struct {
	GuildID snowflake.ID
}

func NewCurrentTrackChangedEvent(
	guildID snowflake.ID,
) CurrentTrackChangedEvent {
	return CurrentTrackChangedEvent{
		GuildID: guildID,
	}
}

func (e CurrentTrackChangedEvent) isEvent() {}
