use anyhow::{Context as _, Result};

use super::{MODULE_NAME, state::TestState};
use crate::{Command, Context, modules::ModuleError};

#[poise::command(slash_command, guild_only)]
pub async fn test(ctx: Context<'_>) -> Result<()> {
    let count = match ctx.data().get::<TestState>() {
        Some(state) => {
            let mut lock = state
                .count
                .lock()
                .map_err(|e| anyhow::anyhow!(e.to_string()))
                .context(format!(
                    "failed to acquire lock in `{}` command",
                    MODULE_NAME
                ))?;
            *lock += 1;
            *lock
        }
        None => {
            return Err(ModuleError::StateNotRegistered {
                module_name: MODULE_NAME.to_owned(),
            }
            .into());
        }
    };

    let ordinal_suffix = {
        if (count / 10) == 1 {
            "th"
        } else {
            match count % 10 {
                1 => "st",
                2 => "nd",
                3 => "rd",
                _ => "th",
            }
        }
    };

    ctx.say(format!(
        "This is a test message and the {}{} response since the last restart.",
        count, ordinal_suffix
    ))
    .await?;

    Ok(())
}

pub fn all() -> Vec<Command> {
    vec![test()]
}
