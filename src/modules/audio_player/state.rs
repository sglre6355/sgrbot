use std::sync::Arc;

use lavalink_rs::client::LavalinkClient;

pub struct AudioPlayerState {
    pub lavalink: Arc<LavalinkClient>,
}
