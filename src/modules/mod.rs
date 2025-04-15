mod error;

#[cfg(feature = "test")]
mod test;

use anyhow::Result;
use async_trait::async_trait;
use error::ModuleError;
use poise::{Framework, FrameworkContext, FrameworkOptions};
use serenity::all::{ClientBuilder, Context as SerenityContext, FullEvent, Ready};
use tracing::info;

use crate::StateStore;

#[async_trait]
pub trait Module {
    fn configure_framework_options(
        &self,
        options: &mut FrameworkOptions<StateStore, anyhow::Error>,
    );

    async fn setup(
        &self,
        state_store: &StateStore,
        ctx: &SerenityContext,
        ready: &Ready,
        framework: &Framework<StateStore, anyhow::Error>,
    ) -> Result<(), anyhow::Error> {
        let _ = (state_store, ctx, ready, framework);

        Ok(())
    }

    fn configure_client(&self, builder: ClientBuilder) -> ClientBuilder {
        builder
    }

    async fn handle_event(
        &self,
        ctx: &SerenityContext,
        event: &FullEvent,
        framework: FrameworkContext<'_, StateStore, anyhow::Error>,
        data: &StateStore,
    ) -> Result<()> {
        let _ = (ctx, event, framework, data);

        Ok(())
    }
}

inventory::collect!(&'static (dyn Module + Sync));

pub fn configure_framework_options(options: &mut FrameworkOptions<StateStore, anyhow::Error>) {
    let modules = inventory::iter::<&'static (dyn Module + Sync)>.into_iter();

    for module in modules.clone() {
        module.configure_framework_options(options);
    }

    info!("Registered {} enabled module(s)", modules.count());
}

pub fn configure_client(mut builder: ClientBuilder) -> ClientBuilder {
    let modules = inventory::iter::<&'static (dyn Module + Sync)>.into_iter();

    for module in modules {
        builder = module.configure_client(builder);
    }

    info!("Applied client configuration for enabled module(s)");

    builder
}

pub async fn setup_enabled(
    state_store: &StateStore,
    ctx: &SerenityContext,
    ready: &Ready,
    framework: &Framework<StateStore, anyhow::Error>,
) -> Result<(), anyhow::Error> {
    let modules = inventory::iter::<&'static (dyn Module + Sync)>.into_iter();

    for module in modules {
        module.setup(state_store, ctx, ready, framework).await?
    }

    info!("Applied client configuration for enabled module(s)");

    Ok(())
}

pub async fn event_handler(
    ctx: &SerenityContext,
    event: &FullEvent,
    framework: FrameworkContext<'_, StateStore, anyhow::Error>,
    data: &StateStore,
) -> Result<()> {
    let modules = inventory::iter::<&'static (dyn Module + Sync)>.into_iter();

    for module in modules {
        module.handle_event(ctx, event, framework, data).await?;
    }

    Ok(())
}
