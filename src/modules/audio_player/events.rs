use std::time::Duration;

use anyhow::Result;
use lavalink_rs::{
    client::LavalinkClient,
    hook,
    model::events::{Ready, TrackEnd, TrackStart},
};
use poise::FrameworkContext;
use serenity::all::{Context as SerenityContext, CreateMessage, FullEvent};
use tracing::{debug, info};

use super::{
    errors::SongbirdError,
    logic::{create_now_playing_embed, get_lavalink_client, leave_voice_channel},
    models::PlayerContextData,
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

        let call = match manager.get(guild_id) {
            Some(call) => call,
            None => return Ok(()),
        };
        let channel_id = {
            let lock = call.lock().await;
            match lock.current_channel() {
                Some(id) => id,
                None => return Ok(()),
            }
        };

        let user_count_in_channel = ctx
            .cache
            .guild(guild_id)
            .expect("`VoiceStateUpdate` events should only be dispatched from guild voice channels")
            .voice_states
            .values()
            .filter(|vs| vs.channel_id.map(songbird::id::ChannelId::from) == Some(channel_id))
            .count();

        if user_count_in_channel <= 1 {
            let player_context = match lavalink_client.get_player_context(guild_id) {
                Some(player_context) => player_context,
                None => return Ok(()),
            };
            let now_playing = player_context.get_player().await?.track;

            if now_playing.is_some() {
                player_context.get_queue().clear()?;
                player_context.skip()?;
            }

            // wait for track end event to dispatch
            tokio::time::sleep(Duration::from_secs(5)).await;

            leave_voice_channel(manager, lavalink_client.clone(), guild_id).await?;
        }
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

    let now_playing_embed = {
        let lock = data.now_playing_embed.lock().await;
        lock.clone()
    };

    if let Some(message) = now_playing_embed {
        message.delete(data.http.clone()).await.unwrap();
    }

    let track = event.track.clone();
    let embed = create_now_playing_embed(track);
    let message = CreateMessage::new().embed(embed);

    let message = {
        let lock = *data.channel_id.lock().await;

        lock.send_message(data.http.clone(), message).await.unwrap()
    };

    let mut lock = data.now_playing_embed.lock().await;
    *lock = Some(message);
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

    let now_playing_embed = {
        let lock = data.now_playing_embed.lock().await;
        lock.clone()
    };

    if let Some(message) = now_playing_embed {
        message.delete(data.http.clone()).await.unwrap();
    }

    let mut lock = data.now_playing_embed.lock().await;
    *lock = None;
}
