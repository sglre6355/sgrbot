mod autocompletes;
mod commands;
mod errors;
mod events;
mod logic;
mod models;
mod state;

use std::{env, sync::Arc};

use anyhow::Result;
use async_trait::async_trait;
use lavalink_rs::{
    client::LavalinkClient, model::events::Events, node::NodeBuilder,
    prelude::NodeDistributionStrategy,
};
use poise::{Framework, FrameworkContext, FrameworkOptions};
use serenity::all::{ClientBuilder, Context as SerenityContext, FullEvent, Ready};
use songbird::SerenityInit as _;
use state::AudioPlayerState;
use tracing::info;

use super::Module;
use crate::StateStore;

pub const MODULE_NAME: &str = "audio-player";

pub struct AudioPlayerModule;

#[async_trait]
impl Module for AudioPlayerModule {
    fn configure_framework_options(
        &self,
        options: &mut FrameworkOptions<StateStore, anyhow::Error>,
    ) {
        options.commands.extend(commands::all());
    }

    async fn setup(
        &self,
        state_store: &StateStore,
        ctx: &SerenityContext,
        _ready: &Ready,
        _framework: &Framework<StateStore, anyhow::Error>,
    ) -> Result<(), anyhow::Error> {
        let events = Events {
            raw: Some(events::raw_event),
            ready: Some(events::ready_event),
            track_start: Some(events::track_start),
            track_end: Some(events::track_end),
            ..Default::default()
        };

        let node = NodeBuilder {
            hostname: env::var("LAVALINK_ADDRESS")
                .expect("`LAVALINK_ADDRESS` environmental variable"),
            is_ssl: env::var("LAVALINK_SSL")
                .map(|s| ["true", "1"].contains(&s.to_lowercase().as_str()))
                .unwrap_or(false),
            events: Events::default(),
            password: env::var("LAVALINK_PASSWORD")
                .expect("`LAVALINK_PASSWORD` environmental variable"),
            user_id: ctx.cache.current_user().id.into(),
            session_id: None,
        };

        info!("Lavalink node configuration loaded");

        let client =
            LavalinkClient::new(events, vec![node], NodeDistributionStrategy::round_robin()).await;

        state_store.insert(Arc::new(AudioPlayerState {
            lavalink: Arc::new(client),
        }));

        Ok(())
    }

    fn configure_client(&self, builder: ClientBuilder) -> ClientBuilder {
        builder.register_songbird()
    }

    async fn handle_event(
        &self,
        ctx: &SerenityContext,
        event: &FullEvent,
        framework: FrameworkContext<'_, StateStore, anyhow::Error>,
        data: &StateStore,
    ) -> Result<()> {
        events::handler(ctx, event, framework, data).await?;

        Ok(())
    }
}

inventory::submit! {
    &AudioPlayerModule as &(dyn Module + Sync)
}
