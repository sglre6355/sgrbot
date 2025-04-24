use std::sync::Arc;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serenity::all::{ChannelId, Color, Http, Message};
use tokio::sync::Mutex;

#[derive(Debug)]
pub struct PlayerContextData {
    pub channel_id: Mutex<ChannelId>,
    pub http: Arc<Http>,
    pub now_playing_embed: Mutex<Option<Message>>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct TrackUserData {
    pub requester_name: String,
    pub requester_avatar_url: String,
    pub request_timestamp: DateTime<Utc>,
}

#[derive(Debug, PartialEq)]
pub enum Source {
    Youtube,
    Spotify,
    Soundcloud,
    Twitch,
    Other,
}

impl Source {
    pub fn from_source_name(name: String) -> Self {
        match name.as_str() {
            "youtube" => Self::Youtube,
            "spotify" => Self::Spotify,
            "soundcloud" => Self::Soundcloud,
            "twitch" => Self::Twitch,
            _ => Self::Other,
        }
    }

    pub fn color(&self) -> Color {
        // source: https://brandfetch.com/
        match self {
            Self::Youtube => Color::new(0xff0000),
            Self::Spotify => Color::new(0x1ed760),
            Self::Soundcloud => Color::new(0xff5500),
            Self::Twitch => Color::new(0x9147ff),
            Self::Other => Color::default(),
        }
    }

    pub fn icon_url(&self) -> &str {
        // source: https://brandfetch.com/
        match self {
            Self::Youtube => {
                "https://cdn.brandfetch.io/idVfYwcuQz/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
            }
            Self::Spotify => {
                "https://cdn.brandfetch.io/id20mQyGeY/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
            }
            Self::Soundcloud => {
                "https://cdn.brandfetch.io/id3ytDFop3/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
            }
            Self::Twitch => {
                "https://cdn.brandfetch.io/idIwZCwD2f/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
            }
            Self::Other => "https://cdn3.iconfinder.com/data/icons/iconpark-vol-2/48/play-256.png",
        }
    }
}
