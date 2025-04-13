mod event_handler;
mod modules;
mod state_store;

use std::env;

use anyhow::Result;
use event_handler::DefaultHandler;
use poise::FrameworkOptions;
use serenity::prelude::{Client, GatewayIntents};
use state_store::StateStore;
use tracing::{info, instrument};

pub const VERSION: &str = env!("CARGO_PKG_VERSION");

pub type Command = poise::Command<StateStore, anyhow::Error>;
pub type Context<'a> = poise::Context<'a, StateStore, anyhow::Error>;

#[tokio::main]
#[instrument]
async fn main() -> Result<()> {
    // enable logging with tracing
    tracing_subscriber::fmt::init();

    info!("Starting sgrbot v{}", VERSION);

    let state_store = StateStore::default();
    let mut options: FrameworkOptions<StateStore, anyhow::Error> = FrameworkOptions {
        event_handler: |ctx, event, framework, data| {
            Box::pin(modules::event_handler(ctx, event, framework, data))
        },
        ..Default::default()
    };
    modules::register_enabled(&mut options, &state_store);

    let framework = poise::Framework::builder()
        .options(options)
        .setup(|ctx, _ready, framework| {
            Box::pin(async move {
                poise::builtins::register_globally(ctx, &framework.options().commands).await?;
                Ok(state_store)
            })
        })
        .build();

    let token = env::var("DISCORD_TOKEN").expect("`DISCORD_TOKEN` environment variable");
    let intents = GatewayIntents::non_privileged() | GatewayIntents::MESSAGE_CONTENT;

    let mut builder = Client::builder(token, intents)
        .framework(framework)
        .event_handler(DefaultHandler);

    modules::configure_client(&mut builder);

    let mut client = builder.await?;
    client.start().await?;

    Ok(())
}
