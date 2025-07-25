use std::time::Duration;

use anyhow::Result;
use chrono::Utc;
use lavalink_rs::prelude::{SearchEngines, TrackInQueue, TrackLoadData};
use poise::CreateReply;
use serenity::all::{Channel, Color, CreateEmbed};

use super::{
    autocompletes::{autocomplete_search_query, autocomplete_track_number},
    errors::{JoinError, LeaveError, SongbirdError},
    logic::{
        create_queue_embed, get_lavalink_client, join_voice_channel, leave_voice_channel,
        resolve_target_voice_channel_id, set_now_playing_text_channel,
    },
    models::{PlayerContextData, TrackUserData},
};
use crate::{Command, Context};

#[poise::command(slash_command, guild_only)]
pub async fn join(
    ctx: Context<'_>,
    #[channel_types("Voice")] voice_channel: Option<Channel>,
) -> Result<()> {
    let guild_id = ctx
        .guild_id()
        .expect("this command should only be run in guilds");

    let voice_states = ctx
        .guild()
        .expect("this command should only be run in guilds")
        .voice_states
        .clone();

    let voice_channel_id = {
        match resolve_target_voice_channel_id(voice_channel, &voice_states, &ctx.author().id) {
            Ok(id) => id,
            Err(JoinError::MissingTargetVoiceChannel) => {
                let embed = CreateEmbed::new()
                    .description("Join a voice channel or specify one to use this command.")
                    .color(Color::RED);
                let reply = CreateReply::default().embed(embed);

                ctx.send(reply).await?;

                return Ok(());
            }
            Err(others) => return Err(others.into()),
        }
    };

    let manager = songbird::get(ctx.serenity_context())
        .await
        .ok_or(SongbirdError::SongbirdNotRegistered)?;

    let lavalink_client = get_lavalink_client(ctx.data())?;

    join_voice_channel(
        manager,
        lavalink_client,
        ctx.serenity_context().http.clone(),
        guild_id,
        ctx.channel_id(),
        voice_channel_id,
    )
    .await?;

    let embed = CreateEmbed::new()
        .description(format!("Connected to <#{voice_channel_id}>."))
        .color(Color::new(0x08c404));
    let reply = CreateReply::default().embed(embed);

    ctx.send(reply).await?;

    Ok(())
}

#[poise::command(slash_command, guild_only)]
pub async fn leave(ctx: Context<'_>) -> Result<()> {
    ctx.defer().await?;

    let guild_id = ctx
        .guild_id()
        .expect("this command should only be run in guilds");

    let manager = songbird::get(ctx.serenity_context())
        .await
        .ok_or(SongbirdError::SongbirdNotRegistered)?;

    let lavalink_client = get_lavalink_client(ctx.data())?;

    if let Some(player_context) = lavalink_client.get_player_context(guild_id)
        && let now_playing = player_context.get_player().await?.track
        && now_playing.is_some()
    {
        player_context.get_queue().clear()?;
        player_context.skip()?;

        // wait for now playing embed to be deleted before disconnecting from voice channel
        // TODO: detect now_playing_embed is None and proceed?
        tokio::time::sleep(Duration::from_secs(1)).await;
    }

    match leave_voice_channel(manager, lavalink_client, guild_id).await {
        Ok(_) => {}
        Err(LeaveError::NotConnected) => {
            let embed = CreateEmbed::new()
                .description("Not connected to any voice channel.")
                .color(Color::RED);
            let reply = CreateReply::default().embed(embed);

            ctx.send(reply).await?;

            return Ok(());
        }
        Err(others) => return Err(others.into()),
    }

    let embed = CreateEmbed::new()
        .description("Disconnected.")
        .color(Color::new(0x08c404));
    let reply = CreateReply::default().embed(embed);

    ctx.send(reply).await?;

    Ok(())
}

#[poise::command(slash_command, guild_only)]
pub async fn play(
    ctx: Context<'_>,
    #[autocomplete = "autocomplete_search_query"] query: String,
) -> Result<()> {
    ctx.defer().await?;

    let guild_id = ctx
        .guild_id()
        .expect("this command should only be run in guilds");

    let voice_states = ctx
        .guild()
        .expect("this command should only be run in guilds")
        .voice_states
        .clone();

    let voice_channel_id =
        match resolve_target_voice_channel_id(None, &voice_states, &ctx.author().id) {
            Ok(id) => id,
            Err(JoinError::MissingTargetVoiceChannel) => {
                let embed = CreateEmbed::new()
                    .description("Join a voice channel or run `/join` before using this command.")
                    .color(Color::RED);
                let reply = CreateReply::default().embed(embed);

                ctx.send(reply).await?;

                return Ok(());
            }
            Err(others) => return Err(others.into()),
        };

    let manager = songbird::get(ctx.serenity_context())
        .await
        .ok_or(SongbirdError::SongbirdNotRegistered)?;

    let lavalink_client = get_lavalink_client(ctx.data())?;

    join_voice_channel(
        manager,
        lavalink_client.clone(),
        ctx.serenity_context().http.clone(),
        guild_id,
        ctx.channel_id(),
        voice_channel_id,
    )
    .await?;

    let player_context = lavalink_client
        .get_player_context(guild_id)
        .expect("`join_voice_channel` should have initialized player context");

    // FIXME: remove unwrap
    let player_context_data = player_context.data::<PlayerContextData>().unwrap();
    set_now_playing_text_channel(player_context_data, ctx.channel_id()).await;

    let query = {
        if query.starts_with("http") {
            query
        } else {
            SearchEngines::YouTube.to_query(&query)?
        }
    };

    let loaded_tracks = lavalink_client.load_tracks(guild_id, &query).await?;

    let mut playlist_info = None;

    let mut tracks: Vec<TrackInQueue> = match loaded_tracks.data {
        Some(TrackLoadData::Track(x)) => vec![x.into()],
        Some(TrackLoadData::Search(x)) => vec![x[0].clone().into()],
        Some(TrackLoadData::Playlist(x)) => {
            playlist_info = Some(x.info);
            x.tracks.iter().map(|x| x.clone().into()).collect()
        }
        _ => {
            return Ok(());
        }
    };

    let avatar_url = ctx
        .author()
        .avatar_url()
        .unwrap_or(ctx.author().default_avatar_url());

    let track_user_data = TrackUserData {
        requester_name: ctx.author().name.clone(),
        requester_avatar_url: avatar_url,
        request_timestamp: Utc::now(),
    };
    let track_user_data_value = Some(serde_json::to_value(track_user_data)?);

    for i in &mut tracks {
        i.track.user_data = track_user_data_value.clone();
    }

    let queue = player_context.get_queue();
    queue.append(tracks.clone().into())?;

    let description = {
        if let Some(info) = playlist_info {
            format!("Added playlist **{}** to the queue", info.name)
        } else {
            let first = tracks.first().unwrap();
            format!(
                "Added [{}]({}) to the queue.",
                first.clone().track.info.title,
                first.clone().track.info.uri.unwrap(),
            )
        }
    };
    let embed = CreateEmbed::new().description(description);
    let reply = CreateReply::default().embed(embed);

    ctx.send(reply).await?;

    if player_context.get_player().await?.track.is_none()
        && queue.get_track(0).await.is_ok_and(|x| x.is_some())
    {
        player_context.skip()?;
    }

    Ok(())
}

#[poise::command(slash_command, guild_only)]
pub async fn stop(ctx: Context<'_>) -> Result<()> {
    let guild_id = ctx
        .guild_id()
        .expect("this command should only be run in guilds");

    let lavalink_client = get_lavalink_client(ctx.data())?;

    let Some(player_context) = lavalink_client.get_player_context(guild_id) else {
        let embed = CreateEmbed::new().description("Not connected to any voice channel.");
        let reply = CreateReply::default().embed(embed);

        ctx.send(reply).await?;

        return Ok(());
    };

    let now_playing = player_context.get_player().await?.track;

    if now_playing.is_some() {
        player_context.get_queue().clear()?;
        player_context.skip()?;

        let embed = CreateEmbed::new().description("Stopped playback.");
        let reply = CreateReply::default().embed(embed);

        ctx.send(reply).await?;
    } else {
        let embed = CreateEmbed::new().description("Nothing is playing right now.");
        let reply = CreateReply::default().embed(embed);

        ctx.send(reply).await?;
    }

    Ok(())
}

#[poise::command(slash_command, guild_only)]
pub async fn pause(ctx: Context<'_>) -> Result<()> {
    let guild_id = ctx
        .guild_id()
        .expect("this command should only be run in guilds");

    let lavalink_client = get_lavalink_client(ctx.data())?;

    let Some(player_context) = lavalink_client.get_player_context(guild_id) else {
        let embed = CreateEmbed::new().description("Not connected to any voice channel.");
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
        return Ok(());
    };

    let now_playing = player_context.get_player().await?.track;
    if now_playing.is_some() {
        player_context.set_pause(true).await?;

        let embed = CreateEmbed::new().description("Paused playback.");
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
    } else {
        let embed = CreateEmbed::new().description("Nothing is playing right now.");
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
    }

    Ok(())
}

#[poise::command(slash_command, guild_only)]
pub async fn resume(ctx: Context<'_>) -> Result<()> {
    let guild_id = ctx
        .guild_id()
        .expect("this command should only be run in guilds");

    let lavalink_client = get_lavalink_client(ctx.data())?;

    let Some(player_context) = lavalink_client.get_player_context(guild_id) else {
        let embed = CreateEmbed::new().description("Not connected to any voice channel.");
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
        return Ok(());
    };

    let now_playing = player_context.get_player().await?.track;
    if now_playing.is_some() {
        player_context.set_pause(false).await?;

        let embed = CreateEmbed::new().description("Resumed playback.");
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
    } else {
        let embed = CreateEmbed::new().description("Nothing is playing right now.");
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
    }

    Ok(())
}

#[poise::command(slash_command, guild_only)]
pub async fn skip(ctx: Context<'_>) -> Result<()> {
    let guild_id = ctx
        .guild_id()
        .expect("this command should only be run in guilds");

    let lavalink_client = get_lavalink_client(ctx.data())?;

    let Some(player_context) = lavalink_client.get_player_context(guild_id) else {
        let embed = CreateEmbed::new().description("Not connected to any voice channel.");
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
        return Ok(());
    };

    if let Some(now_playing) = player_context.get_player().await?.track {
        player_context.skip()?;

        let embed = CreateEmbed::new().description(format!(
            "Skipped [{}]({}).",
            now_playing.info.title,
            now_playing.info.uri.unwrap()
        ));
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
    } else {
        let embed = CreateEmbed::new().description("Nothing is playing right now.");
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
    }

    Ok(())
}

#[poise::command(slash_command, guild_only, subcommands("list", "remove", "clear"))]
pub async fn queue(_ctx: Context<'_>) -> Result<()> {
    Ok(())
}

#[poise::command(slash_command, guild_only)]
pub async fn list(ctx: Context<'_>, page: Option<usize>) -> Result<()> {
    let guild_id = ctx
        .guild_id()
        .expect("this command should only be run in guilds");

    let lavalink_client = get_lavalink_client(ctx.data())?;

    let Some(player_context) = lavalink_client.get_player_context(guild_id) else {
        let embed = CreateEmbed::new().description("Not connected to any voice channel.");
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
        return Ok(());
    };

    let page = page.map(|n| n - 1).unwrap_or(0);
    let embed = create_queue_embed(player_context.get_queue(), page).await;
    let reply = CreateReply::default().embed(embed).ephemeral(true);
    ctx.send(reply).await?;

    Ok(())
}

#[poise::command(slash_command, guild_only)]
pub async fn remove(
    ctx: Context<'_>,
    #[autocomplete = "autocomplete_track_number"] track_number: usize,
) -> Result<()> {
    let guild_id = ctx
        .guild_id()
        .expect("this command should only be run in guilds");

    let lavalink_client = get_lavalink_client(ctx.data())?;

    let Some(player_context) = lavalink_client.get_player_context(guild_id) else {
        let embed = CreateEmbed::new().description("Not connected to any voice channel.");
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
        return Ok(());
    };

    let queue = player_context.get_queue();

    if !(0 < track_number && track_number <= queue.get_count().await?) {
        let embed = CreateEmbed::new()
            .description("Invalid track number specified.")
            .color(Color::RED);
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
        return Ok(());
    }

    let index = track_number - 1;
    let track_info = queue
        .get_track(index)
        .await?
        .expect("track at given index must exist due to prior bounds check")
        .track
        .info;

    queue.remove(index)?;

    let embed = CreateEmbed::new().description({
        if let Some(uri) = track_info.uri {
            format!("Removed [{}]({}).", track_info.title, uri)
        } else {
            format!("Removed **{}**.", track_info.title)
        }
    });
    let reply = CreateReply::default().embed(embed);
    ctx.send(reply).await?;

    Ok(())
}

#[poise::command(slash_command, guild_only)]
pub async fn clear(ctx: Context<'_>) -> Result<()> {
    let guild_id = ctx
        .guild_id()
        .expect("this command should only be run in guilds");

    let lavalink_client = get_lavalink_client(ctx.data())?;

    let Some(player_context) = lavalink_client.get_player_context(guild_id) else {
        let embed = CreateEmbed::new().description("Not connected to any voice channel.");
        let reply = CreateReply::default().embed(embed);
        ctx.send(reply).await?;
        return Ok(());
    };

    player_context.get_queue().clear()?;

    let embed = CreateEmbed::new().description("Cleared the queue.");
    let reply = CreateReply::default().embed(embed);
    ctx.send(reply).await?;

    Ok(())
}

pub fn all() -> Vec<Command> {
    vec![
        join(),
        leave(),
        play(),
        stop(),
        pause(),
        resume(),
        skip(),
        queue(),
    ]
}
