use std::fmt::{Debug, Display};

use poise::{CreateReply, FrameworkError};
use serenity::all::{Color, CreateAllowedMentions, CreateEmbed};
use tracing::{error, warn};

// This function was excerpted and modified from Poise (https://github.com/serenity-rs/poise)
// Copyright (c) 2021 kangalioo. Licensed under the MIT License.
pub async fn on_error<U, E: Display + Debug>(
    error: FrameworkError<'_, U, E>,
) -> Result<(), serenity::Error> {
    match error {
        FrameworkError::Setup { error, .. } => {
            error!("Error in user data setup: {}", error);
        }
        FrameworkError::EventHandler { error, event, .. } => error!(
            "User event event handler encountered an error on {} event: {}",
            event.snake_case_name(),
            error
        ),
        FrameworkError::Command { ctx, error, .. } => {
            let error = error.to_string();
            error!("An error occured in a command: {}", error);

            let embed = CreateEmbed::new()
                .title("Command Error")
                .description(error)
                .color(Color::DARK_RED);
            let reply = CreateReply::default().embed(embed);

            ctx.send(reply).await?;
        }
        FrameworkError::SubcommandRequired { ctx } => {
            let subcommands = ctx
                .command()
                .subcommands
                .iter()
                .map(|s| &*s.name)
                .collect::<Vec<_>>();

            let embed = CreateEmbed::new()
                .title("Subcommand Required")
                .description(format!(
                    "You must specify one of the following subcommands: `{}`",
                    subcommands.join(", ")
                ))
                .color(Color::DARK_RED);
            let reply = CreateReply::default().embed(embed);

            ctx.send(reply.ephemeral(true)).await?;
        }
        FrameworkError::CommandPanic {
            ctx, payload: _, ..
        } => {
            // Not showing the payload to the user because it may contain sensitive info
            let embed = CreateEmbed::default()
                .title("Internal error")
                .description("An unexpected internal error has occurred.")
                .color(Color::DARK_RED);
            let reply = CreateReply::default().embed(embed);

            ctx.send(reply.ephemeral(true)).await?;
        }
        FrameworkError::ArgumentParse {
            ctx, input, error, ..
        } => {
            // If we caught an argument parse error, give a helpful error message with the
            // command explanation if available
            let usage = match &ctx.command().help_text {
                Some(help_text) => &**help_text,
                None => "Please check the help menu for usage information",
            };
            let description = if let Some(input) = input {
                format!(
                    "**Cannot parse `{}` as argument: {}**\n{}",
                    input, error, usage
                )
            } else {
                format!("**{}**\n{}", error, usage)
            };

            let mentions = CreateAllowedMentions::new()
                .everyone(false)
                .all_roles(false)
                .all_users(false);

            let embed = CreateEmbed::new()
                .title("Invalid Command Argument")
                .description(description)
                .color(Color::DARK_RED);
            let reply = CreateReply::default()
                .embed(embed)
                .allowed_mentions(mentions);

            ctx.send(reply).await?;
        }
        FrameworkError::CommandStructureMismatch {
            ctx, description, ..
        } => {
            error!(
                "Error: failed to deserialize interaction arguments for `/{}`: {}",
                ctx.command.name, description,
            );
        }
        FrameworkError::CommandCheckFailed { ctx, error, .. } => {
            error!(
                "A command check failed in command {} for user {}: {:?}",
                ctx.command().name,
                ctx.author().name,
                error,
            );
        }
        FrameworkError::CooldownHit {
            remaining_cooldown,
            ctx,
            ..
        } => {
            let embed = CreateEmbed::new()
                .title("Cooldown Active")
                .description(format!(
                    "You're too fast. Please wait {} seconds before retrying",
                    remaining_cooldown.as_secs()
                ))
                .color(Color::DARK_RED);
            let reply = CreateReply::default().embed(embed);

            ctx.send(reply.ephemeral(true)).await?;
        }
        FrameworkError::MissingBotPermissions {
            missing_permissions,
            ctx,
            ..
        } => {
            let embed = CreateEmbed::new()
                .title("Missing Bot Permissions")
                .description(format!(
                    "Command cannot be executed because the bot is lacking permissions: {}",
                    missing_permissions,
                ))
                .color(Color::DARK_RED);
            let reply = CreateReply::default().embed(embed);

            ctx.send(reply.ephemeral(true)).await?;
        }
        FrameworkError::MissingUserPermissions {
            missing_permissions,
            ctx,
            ..
        } => {
            let description = if let Some(missing_permissions) = missing_permissions {
                format!(
                    "You're lacking permissions for `{}{}`: {}",
                    ctx.prefix(),
                    ctx.command().name,
                    missing_permissions,
                )
            } else {
                format!(
                    "You may be lacking permissions for `{}{}`. Not executing for safety",
                    ctx.prefix(),
                    ctx.command().name,
                )
            };

            let embed = CreateEmbed::new()
                .title("Permission Denied")
                .title("Missing User Permissions")
                .description(description)
                .color(Color::DARK_RED);
            let reply = CreateReply::default().embed(embed);

            ctx.send(reply.ephemeral(true)).await?;
        }
        FrameworkError::NotAnOwner { ctx, .. } => {
            let embed = CreateEmbed::new()
                .title("Permission Denied")
                .description("Only bot owners can call this command")
                .color(Color::DARK_RED);
            let reply = CreateReply::default().embed(embed);
            ctx.send(reply.ephemeral(true)).await?;
        }
        FrameworkError::GuildOnly { ctx, .. } => {
            let embed = CreateEmbed::new()
                .title("Guild-Only Command")
                .description("You cannot run this command in DMs.")
                .color(Color::DARK_RED);
            let reply = CreateReply::default().embed(embed);
            ctx.send(reply.ephemeral(true)).await?;
        }
        FrameworkError::DmOnly { ctx, .. } => {
            let embed = CreateEmbed::new()
                .title("DM-Only Command")
                .description("You cannot run this command outside DMs.")
                .color(Color::DARK_RED);
            let reply = CreateReply::default().embed(embed);
            ctx.send(reply.ephemeral(true)).await?;
        }
        FrameworkError::NsfwOnly { ctx, .. } => {
            let embed = CreateEmbed::new()
                .title("NSFW-Only Command")
                .description("You cannot run this command outside NSFW channels.")
                .color(Color::DARK_RED);
            let reply = CreateReply::default().embed(embed);
            ctx.send(reply.ephemeral(true)).await?;
        }
        FrameworkError::DynamicPrefix { error, msg, .. } => {
            error!(
                "Dynamic prefix failed for message {:?}: {}",
                msg.content, error
            );
        }
        FrameworkError::UnknownCommand {
            msg_content,
            prefix,
            ..
        } => {
            warn!(
                "Recognized prefix `{}`, but didn't recognize command name in `{}`",
                prefix, msg_content,
            );
        }
        FrameworkError::UnknownInteraction { interaction, .. } => {
            warn!("received unknown interaction \"{}\"", interaction.data.name);
        }
        FrameworkError::__NonExhaustive(unreachable) => match unreachable {},
    }

    Ok(())
}
