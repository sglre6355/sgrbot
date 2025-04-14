mod commands;
mod events;
mod state;

use std::sync::{Arc, Mutex};

use anyhow::Result;
use async_trait::async_trait;
use poise::{Framework, FrameworkContext, FrameworkOptions};
use serenity::all::{Context as SerenityContext, FullEvent, Ready};
use state::TestState;

use super::Module;
use crate::StateStore;

pub struct TestModule;

#[async_trait]
impl Module for TestModule {
    fn configure_framework_options(
        &self,
        options: &mut FrameworkOptions<StateStore, anyhow::Error>,
    ) {
        options.commands.extend(commands::all());
    }

    async fn setup(
        &self,
        state_store: &StateStore,
        _ctx: &SerenityContext,
        _ready: &Ready,
        _framework: &Framework<StateStore, anyhow::Error>,
    ) -> Result<(), anyhow::Error> {
        state_store.insert(Arc::new(TestState {
            count: Mutex::new(0),
        }));

        Ok(())
    }

    async fn handle_event(
        &self,
        ctx: &SerenityContext,
        event: &FullEvent,
        framework: FrameworkContext<'_, StateStore, anyhow::Error>,
        data: &StateStore,
    ) -> Result<()> {
        events::handler(ctx, event, framework, data).await?;

        Ok(())
    }
}

inventory::submit! {
    &TestModule as &(dyn Module + Sync)
}
