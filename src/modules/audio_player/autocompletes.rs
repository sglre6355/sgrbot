use lavalink_rs::prelude::SearchEngines;
use serenity::all::AutocompleteChoice;
use tracing::error;

use super::logic::{get_lavalink_client, search_tracks};
use crate::Context;

pub async fn autocomplete_search_query<'a>(
    ctx: Context<'_>,
    partial: &str,
) -> impl Iterator<Item = String> + Send + 'a {
    let lavalink_client = match get_lavalink_client(ctx.data()) {
        Ok(client) => client,
        Err(error) => {
            error!("autocomplete callback failed: {}", error);
            return Vec::new().into_iter().take(0);
        }
    };

    let search_result: Vec<String> = search_tracks(
        lavalink_client,
        ctx.guild_id()
            .expect("this autocomplete callback should only be used with guild-only commands"),
        SearchEngines::YouTube,
        partial,
    )
    .await
    .unwrap_or(Vec::new())
    .iter()
    .map(|track_info| track_info.title.to_owned())
    .collect();

    search_result.into_iter().take(10)
}

pub async fn autocomplete_track_number<'a>(
    ctx: Context<'_>,
    partial: &str,
) -> impl Iterator<Item = AutocompleteChoice> + Send + 'a {
    let guild_id = ctx
        .guild_id()
        .expect("this autocomplete should only be used with guild-only commands");

    let lavalink_client = match get_lavalink_client(ctx.data()) {
        Ok(client) => client,
        Err(error) => {
            error!("autocomplete callback failed: {}", error);
            return Vec::new().into_iter().take(0);
        }
    };

    let Some(player_context) = lavalink_client.get_player_context(guild_id) else {
        return Vec::new().into_iter().take(0);
    };

    let Ok(queue) = player_context.get_queue().get_queue().await else {
        return Vec::new().into_iter().take(0);
    };

    let choices: Vec<AutocompleteChoice> = queue
        .iter()
        .enumerate()
        .filter_map(|(index, track_in_queue)| {
            let track_number = index + 1;
            let title = &track_in_queue.track.info.title;
            let label = format!("{}. {}", track_number, title);

            if track_number.to_string().starts_with(partial)
                || title.to_lowercase().starts_with(&partial.to_lowercase())
            {
                Some(AutocompleteChoice::new(label, track_number))
            } else {
                None
            }
        })
        .collect();

    // Discord limits autocomplete suggestions to a maximum of 25 choices
    choices.into_iter().take(25)
}
