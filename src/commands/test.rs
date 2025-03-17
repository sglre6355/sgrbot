use anyhow::Result;

use super::{Command, Context};

#[poise::command(slash_command)]
pub async fn ping(ctx: Context<'_>) -> Result<()> {
    let response = "Pong!".to_string();
    ctx.say(response).await?;
    Ok(())
}

#[poise::command(slash_command)]
pub async fn greeting(ctx: Context<'_>) -> Result<()> {
    let sender_name = &ctx.author().name;
    let response = format!("Hi, {}! How are you doing today?", sender_name);
    ctx.say(response).await?;
    Ok(())
}

pub fn commands() -> Vec<Command> {
    vec![ping(), greeting()]
}
