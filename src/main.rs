mod error_handler;
mod event_handler;
mod modules;
mod state_store;

use std::env;

use anyhow::Result;
use event_handler::DefaultHandler;
use poise::FrameworkOptions;
use serenity::prelude::{Client, GatewayIntents};
use state_store::StateStore;
use tokio::signal::unix::{SignalKind, signal};
use tracing::{error, info, instrument};

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
        on_error: |error| {
            Box::pin(async move {
                if let Err(error) = error_handler::on_error(error).await {
                    error!("Error handling failed: {}", error);
                }
            })
        },
        ..Default::default()
    };
    modules::configure_framework_options(&mut options);

    let framework = poise::Framework::builder()
        .options(options)
        .setup(|ctx, ready, framework| {
            Box::pin(async move {
                poise::builtins::register_globally(ctx, &framework.options().commands).await?;
                modules::setup_enabled(&state_store, ctx, ready, framework).await?;
                Ok(state_store)
            })
        })
        .build();

    let token = env::var("DISCORD_TOKEN").expect("`DISCORD_TOKEN` environment variable");
    let intents = GatewayIntents::non_privileged() | GatewayIntents::MESSAGE_CONTENT;

    let builder = Client::builder(token, intents)
        .framework(framework)
        .event_handler(DefaultHandler);

    let builder = modules::configure_client(builder);

    let mut client = builder.await?;

    let mut sigint = signal(SignalKind::interrupt())?;
    let mut sigterm = signal(SignalKind::terminate())?;

    tokio::select!(
        result = client.start() => {
            result?
        },
        _ = sigint.recv() => {
            info!("Received SIGINT, terminating...");
        },
        _ = sigterm.recv() => {
            info!("Received SIGTERM, terminating...");
        }
    );

    Ok(())
}
