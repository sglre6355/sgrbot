use std::{collections::HashMap, sync::Arc};

use lavalink_rs::{
    model::track::{TrackData, TrackInfo},
    prelude::{LavalinkClient, SearchEngines, TrackLoadData},
};
use serenity::all::{
    Channel, ChannelId, CreateEmbed, CreateEmbedAuthor, CreateEmbedFooter, GuildId, Http, UserId,
    VoiceState,
};
use songbird::Songbird;
use tokio::sync::Mutex;

use super::{
    MODULE_NAME,
    errors::{JoinError, LeaveError},
    models::{PlayerContextData, Source, TrackUserData},
    state::AudioPlayerState,
};
use crate::{modules::error::ModuleError, state_store::StateStore};

pub async fn set_now_playing_text_channel(
    player_context_data: Arc<PlayerContextData>,
    text_channel_id: ChannelId,
) {
    let mut lock = player_context_data.channel_id.lock().await;
    *lock = text_channel_id;
}

pub fn get_lavalink_client(state_store: &StateStore) -> Result<Arc<LavalinkClient>, ModuleError> {
    match state_store.get::<AudioPlayerState>() {
        Some(state) => Ok(Arc::clone(&state.lavalink)),
        None => Err(ModuleError::StateNotRegistered {
            module_name: MODULE_NAME.to_owned(),
        }),
    }
}

pub fn resolve_target_voice_channel_id(
    voice_channel: Option<Channel>,
    voice_states: &HashMap<UserId, VoiceState>,
    author_id: &UserId,
) -> Result<ChannelId, JoinError> {
    if let Some(voice_channel) = voice_channel {
        return Ok(voice_channel.id());
    }

    let id = voice_states
        .get(author_id)
        .and_then(|voice_state| voice_state.channel_id)
        .ok_or(JoinError::MissingTargetVoiceChannel)?;

    Ok(id)
}

pub async fn join_voice_channel(
    manager: Arc<Songbird>,
    lavalink_client: Arc<LavalinkClient>,
    http: Arc<Http>,
    guild_id: GuildId,
    text_channel_id: ChannelId,
    voice_channel_id: ChannelId,
) -> Result<(), JoinError> {
    let (connection_info, _) = manager.join_gateway(guild_id, voice_channel_id).await?;

    // FIXME: lavalink-rs incompatible with v0.5
    // this is an ad-hoc patch
    let connection_info = lavalink_rs::model::player::ConnectionInfo {
        endpoint: connection_info.endpoint,
        token: connection_info.token,
        session_id: connection_info.session_id,
    };

    lavalink_client
        .create_player_context_with_data::<PlayerContextData>(
            guild_id,
            connection_info,
            Arc::new(PlayerContextData {
                channel_id: Mutex::new(text_channel_id),
                http,
                now_playing_embed: Mutex::new(None),
            }),
        )
        .await?;

    Ok(())
}

pub async fn leave_voice_channel(
    manager: Arc<Songbird>,
    lavalink_client: Arc<LavalinkClient>,
    guild_id: GuildId,
) -> Result<(), LeaveError> {
    lavalink_client.delete_player(guild_id).await?;

    if manager.get(guild_id).is_none() {
        return Err(LeaveError::NotConnected);
    }

    match manager.remove(guild_id).await {
        Ok(_) => Ok(()),
        Err(error) => Err(error.into()),
    }
}

pub fn format_track_length_ms(milliseconds: u64) -> String {
    let total_seconds = milliseconds / 1000;
    let hours = total_seconds / 3600;
    let minutes = (total_seconds % 3600) / 60;
    let seconds = total_seconds % 60;

    let mut parts = Vec::new();
    if hours > 0 {
        parts.push(format!("{}h", hours));
    }
    if minutes > 0 {
        parts.push(format!("{}m", minutes));
    }
    if seconds > 0 {
        parts.push(format!("{}s", seconds));
    }

    parts.join(" ")
}

pub fn create_now_playing_embed(track: TrackData) -> CreateEmbed {
    let user_data: TrackUserData = serde_json::from_str(
        &track
            .user_data
            .expect("user data should be set")
            .to_string(),
    )
    .expect("encoded user data should be valid");

    let source = Source::from_source_name(track.info.source_name);

    let author = CreateEmbedAuthor::new("Now Playing").icon_url(source.icon_url());
    let footer = CreateEmbedFooter::new(format!("Requested by {}", user_data.requester_name))
        .icon_url(user_data.requester_avatar_url);

    let mut embed = CreateEmbed::new()
        .title(track.info.title)
        .color(source.color())
        .author(author)
        .field("Author", track.info.author, true)
        .footer(footer)
        .timestamp(user_data.request_timestamp);

    if let Some(uri) = track.info.uri {
        embed = embed.url(uri);
    }

    if let Some(mut image_url) = track.info.artwork_url {
        // TODO
        if source == Source::Youtube {
            image_url = image_url
                .replace("/sddefault", "/maxresdefault")
                .replace("/hqdefault", "/maxresdefault")
                .replace("/mqdefault", "/maxresdefault")
                .replace("/default", "/maxresdefault");
        }
        if source == Source::Twitch {
            image_url = image_url.replace("440x248", "1280x720");
        }

        embed = embed.image(image_url);
    }

    if !track.info.is_stream {
        embed = embed.field("Duration", format_track_length_ms(track.info.length), true);
    }

    embed
}

pub async fn search_tracks(
    lavalink_client: Arc<LavalinkClient>,
    guild_id: GuildId,
    search_engine: SearchEngines,
    query: &str,
) -> anyhow::Result<Vec<TrackInfo>> {
    let query = search_engine.to_query(query)?;

    let search_result: Vec<TrackInfo> =
        match lavalink_client.load_tracks(guild_id, &query).await?.data {
            Some(TrackLoadData::Search(tracks)) => {
                tracks.iter().map(|track| track.info.to_owned()).collect()
            }
            _ => return Ok(Vec::new()),
        };

    Ok(search_result)
}
