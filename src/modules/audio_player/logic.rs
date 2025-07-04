use std::{collections::HashMap, sync::Arc};

use futures::{StreamExt as _, future};
use lavalink_rs::{
    model::track::{TrackData, TrackInfo},
    player_context::QueueRef,
    prelude::{LavalinkClient, SearchEngines, TrackLoadData},
};
use reqwest::{Client, StatusCode};
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
        parts.push(format!("{hours}h"));
    }
    if minutes > 0 {
        parts.push(format!("{minutes}m"));
    }
    if seconds > 0 {
        parts.push(format!("{seconds}s"));
    }

    parts.join(" ")
}

async fn get_best_thumbnail(track_info: TrackInfo) -> Option<String> {
    let source = Source::from_source_name(track_info.source_name);

    let client = Client::new();

    match source {
        Source::Youtube => {
            let qualities = ["maxresdefault", "sddefault", "hqdefault", "mqdefault"];

            let mut resolved_url = None;

            for quality in &qualities {
                let url = format!(
                    "https://img.youtube.com/vi/{}/{}.jpg",
                    track_info.identifier, quality
                );
                match client.head(&url).send().await {
                    Ok(response) if response.status() == StatusCode::OK => {
                        resolved_url = Some(url);
                        break;
                    }
                    _ => continue,
                }
            }

            resolved_url.or(track_info.artwork_url)
        }
        Source::Twitch => {
            let url = track_info
                .artwork_url
                .clone()
                .expect("Twitch source should always have an artwork_url")
                .replace("440x248", "1280x720");

            match client.head(&url).send().await {
                Ok(response) if response.status() == StatusCode::OK => Some(url),
                _ => track_info.artwork_url,
            }
        }
        _ => track_info.artwork_url,
    }
}

pub async fn create_now_playing_embed(track: TrackData) -> CreateEmbed {
    let user_data: TrackUserData = serde_json::from_str(
        &track
            .user_data
            .expect("user data should be set")
            .to_string(),
    )
    .expect("encoded user data should be valid");

    let source = Source::from_source_name(track.info.source_name.clone());

    let author = CreateEmbedAuthor::new("Now Playing").icon_url(source.icon_url());
    let footer = CreateEmbedFooter::new(format!("Requested by {}", user_data.requester_name))
        .icon_url(user_data.requester_avatar_url);

    let mut embed = CreateEmbed::new()
        .title(track.info.title.clone())
        .color(source.color())
        .author(author)
        .field("Author", track.info.author.clone(), true)
        .footer(footer)
        .timestamp(user_data.request_timestamp);

    if let Some(uri) = track.info.uri.clone() {
        embed = embed.url(uri);
    }

    if let Some(thumbnail_url) = get_best_thumbnail(track.info.clone()).await {
        embed = embed.image(thumbnail_url);
    }

    if !track.info.is_stream {
        embed = embed.field("Duration", format_track_length_ms(track.info.length), true);
    }

    embed
}

pub async fn create_queue_embed(queue: QueueRef, page: usize) -> CreateEmbed {
    const TRACKS_PER_PAGE: usize = 10;

    let track_count = queue.get_count().await.expect(
        "this function should only be called when the bot is connected to a voice channel.",
    );
    let total_pages = track_count.div_ceil(TRACKS_PER_PAGE);

    // TODO
    let page = if page < total_pages {
        page
    } else {
        total_pages - 1
    };

    let start = page * TRACKS_PER_PAGE;
    let end = (start + TRACKS_PER_PAGE).min(track_count);

    let description = {
        if track_count == 0 {
            "Queue is empty.".to_owned()
        } else {
            queue
                .enumerate()
                .filter(|(index, _)| future::ready(start <= *index && *index < end))
                .map(|(index, x)| {
                    if let Some(uri) = x.track.info.uri {
                        format!(
                            "{}. [{}]({}) - {}",
                            index + 1,
                            x.track.info.title,
                            uri,
                            x.track.info.author,
                        )
                    } else {
                        format!(
                            "{}. **{}** - {}",
                            index + 1,
                            x.track.info.title,
                            x.track.info.author,
                        )
                    }
                })
                .collect::<Vec<_>>()
                .await
                .join("\n")
        }
    };

    CreateEmbed::new()
        .title("Queue")
        .description(description)
        .footer(CreateEmbedFooter::new(format!(
            "Page {}/{}",
            page + 1,
            total_pages
        )))
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
