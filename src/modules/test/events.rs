use anyhow::Result;
use poise::FrameworkContext;
use serenity::all::{Context as SerenityContext, FullEvent};

use crate::state_store::StateStore;

pub async fn handler(
    ctx: &SerenityContext,
    event: &FullEvent,
    _framework: FrameworkContext<'_, StateStore, anyhow::Error>,
    _data: &StateStore,
) -> Result<()> {
    if let FullEvent::Message { new_message } = event {
        if new_message.content.contains("TEST EVENT HANDLER") {
            new_message
                .channel_id
                .say(&ctx.http, "Event received")
                .await?;
        }
    }

    Ok(())
}
