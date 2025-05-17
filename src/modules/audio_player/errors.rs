use lavalink_rs::error::LavalinkError;
use songbird::error::JoinError as SongbirdJoinError;
use thiserror::Error;

use crate::modules::ModuleError;

#[derive(Debug, Error)]
pub enum SongbirdError {
    #[error("songbird client is not registered")]
    SongbirdNotRegistered,
    #[error(transparent)]
    JoinError(#[from] SongbirdJoinError),
}

#[derive(Debug, Error)]
pub enum JoinError {
    #[error("unable to determine voice channel to connect")]
    MissingTargetVoiceChannel,
    #[error(transparent)]
    ModuleError(#[from] ModuleError),
    #[error(transparent)]
    SongbirdError(Box<SongbirdError>),
    #[error(transparent)]
    LavalinkError(Box<LavalinkError>),
}

impl From<SongbirdJoinError> for JoinError {
    fn from(value: SongbirdJoinError) -> Self {
        Self::SongbirdError(Box::new(SongbirdError::JoinError(value)))
    }
}

impl From<LavalinkError> for JoinError {
    fn from(value: LavalinkError) -> Self {
        Self::LavalinkError(Box::new(value))
    }
}

#[derive(Debug, Error)]
pub enum LeaveError {
    #[error("bot is not connected to a voice channel in the guild")]
    NotConnected,
    #[error(transparent)]
    ModuleError(#[from] ModuleError),
    #[error(transparent)]
    SongbirdError(#[from] SongbirdError),
    #[error(transparent)]
    LavalinkError(#[from] LavalinkError),
}

impl From<SongbirdJoinError> for LeaveError {
    fn from(value: SongbirdJoinError) -> Self {
        Self::SongbirdError(SongbirdError::JoinError(value))
    }
}
