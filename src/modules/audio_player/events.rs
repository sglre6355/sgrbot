use std::time::Duration;

use anyhow::Result;
use lavalink_rs::{
    client::LavalinkClient,
    hook,
    model::events::{Ready, TrackEnd, TrackStart},
};
use poise::FrameworkContext;
use serenity::all::{Context as SerenityContext, CreateMessage, FullEvent};
use tracing::{debug, info, warn};

use super::{
    errors::SongbirdError,
    logic::{create_now_playing_embed, get_lavalink_client, leave_voice_channel},
    models::{NowPlayingEmbed, PlayerContextData},
};
use crate::state_store::StateStore;

pub async fn handler(
    ctx: &SerenityContext,
    event: &FullEvent,
    _framework: FrameworkContext<'_, StateStore, anyhow::Error>,
    data: &StateStore,
) -> Result<()> {
    let manager = songbird::get(ctx)
        .await
        .ok_or(SongbirdError::SongbirdNotRegistered)?;

    let lavalink_client = get_lavalink_client(data)?;

    if let FullEvent::VoiceStateUpdate { new, .. } = event {
        let guild_id = new.guild_id.expect(
            "`VoiceStateUpdate` events should only be dispatched from guild voice channels",
        );

        let Some(call) = manager.get(guild_id) else {
            return Ok(());
        };
        let Some(channel_id) = call.lock().await.current_channel() else {
            return Ok(());
        };

        let user_count_in_channel = ctx
            .cache
            .guild(guild_id)
            .expect("`VoiceStateUpdate` events should only be dispatched from guild voice channels")
            .voice_states
            .values()
            .filter(|vs| vs.channel_id.map(songbird::id::ChannelId::from) == Some(channel_id))
            .count();

        if user_count_in_channel > 1 {
            return Ok(());
        }

        if let Some(player_context) = lavalink_client.get_player_context(guild_id)
            && let now_playing = player_context.get_player().await?.track
            && now_playing.is_some()
        {
            player_context.get_queue().clear()?;
            player_context.skip()?;

            // wait for track end event to dispatch
            tokio::time::sleep(Duration::from_secs(5)).await;
        }

        leave_voice_channel(manager, lavalink_client.clone(), guild_id).await?;
    }

    Ok(())
}

#[hook]
pub async fn raw_event(_: LavalinkClient, session_id: String, event: &serde_json::Value) {
    if event["op"].as_str() == Some("event") || event["op"].as_str() == Some("playerUpdate") {
        debug!("{:?} -> {:?}", session_id, event);
    }
}

#[hook]
pub async fn ready_event(client: LavalinkClient, session_id: String, event: &Ready) {
    client.delete_all_player_contexts().await.unwrap();
    info!("{:?} -> {:?}", session_id, event);
}

#[hook]
pub async fn track_start(client: LavalinkClient, session_id: String, event: &TrackStart) {
    debug!("{:?} -> {:?}", session_id, event);

    let player_context = client.get_player_context(event.guild_id).unwrap();
    let data = player_context.data::<PlayerContextData>().unwrap();

    let mut lock = data.now_playing_embed.lock().await;

    if let Some(now_playing_embed) = lock.take() {
        if let Err(error) = now_playing_embed.message.delete(data.http.clone()).await {
            warn!("Failed to delete now playing embed: {}", error);
        }
    }

    let track = event.track.clone();
    let embed = create_now_playing_embed(track.clone()).await;
    let message = CreateMessage::new().embed(embed);

    match data
        .channel_id
        .lock()
        .await
        .send_message(data.http.clone(), message)
        .await
    {
        Ok(message) => {
            *lock = Some(NowPlayingEmbed {
                track_identifier: track.info.identifier,
                message,
            })
        }
        Err(error) => warn!("Failed to send now playing embed: {}", error),
    }
}

#[hook]
pub async fn track_end(client: LavalinkClient, session_id: String, event: &TrackEnd) {
    debug!("{:?} -> {:?}", session_id, event);

    let player_context = client
        .get_player_context(event.guild_id)
        .expect("player context should have been initialized when `TrackEnd` event is dispatched");
    let data = player_context
        .data::<PlayerContextData>()
        .expect("player context data should be initialized");

    let mut lock = data.now_playing_embed.lock().await;

    // If now playing message data exists and the track identifier matches that of the event,
    // remove the corresponding message and set the now playing data to None.
    if let Some(ref mut now_playing_embed) = *lock
        && now_playing_embed.track_identifier == event.track.info.identifier
    {
        if let Err(error) = now_playing_embed.message.delete(data.http.clone()).await {
            warn!("Failed to delete now playing embed: {}", error);
        }

        *lock = None;
    }
}
