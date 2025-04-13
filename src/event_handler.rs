use async_trait::async_trait;
use serenity::all::{Context, EventHandler, Ready};
use tracing::{Level, info, instrument};

pub struct DefaultHandler;

#[async_trait]
impl EventHandler for DefaultHandler {
    #[instrument(level = Level::INFO, skip_all)]
    #[instrument(level = Level::DEBUG, skip(self))]
    async fn ready(&self, _ctx: Context, ready: Ready) {
        info!(
            "Connection established: {} ({})",
            ready.user.name, ready.user.id
        );
    }
}
