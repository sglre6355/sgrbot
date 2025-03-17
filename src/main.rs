mod commands;

use std::env;

use anyhow::{bail, Result};
use serenity::{
    async_trait,
    model::gateway::Ready,
    prelude::{Client, Context, EventHandler, GatewayIntents},
};
use tracing::{info, instrument, Level};

struct Handler;

#[async_trait]
impl EventHandler for Handler {
    #[instrument(level = Level::INFO, skip_all)]
    #[instrument(level = Level::DEBUG, skip(self))]
    async fn ready(&self, _ctx: Context, ready: Ready) {
        info!(
            "Connection established: {}({})",
            ready.user.name, ready.user.id
        );
    }
}

#[tokio::main]
#[instrument]
async fn main() -> Result<()> {
    // enable logging with tracing
    tracing_subscriber::fmt::init();

    let token = env::var("DISCORD_TOKEN").expect("`DISCORD_TOKEN` should be in the environment");
    let intents = GatewayIntents::non_privileged();

    let framework = poise::Framework::builder()
        .options(poise::FrameworkOptions {
            commands: commands::commands(),
            ..Default::default()
        })
        .setup(|ctx, _ready, framework| {
            Box::pin(async move {
                poise::builtins::register_globally(ctx, &framework.options().commands).await?;
                Ok(())
            })
        })
        .build();

    let mut client = Client::builder(token, intents)
        .event_handler(Handler)
        .framework(framework)
        .await
        .expect("Failed to initialize the client");

    if let Err(error) = client.start().await {
        bail!("An error occured while starting the client {:?}", error);
    }

    Ok(())
}
