#[cfg(feature = "test")]
mod test;

use anyhow::Result;
use async_trait::async_trait;
use poise::{FrameworkContext, FrameworkOptions};
use serenity::all::{ClientBuilder, Context as SerenityContext, FullEvent};
use tracing::info;

use crate::StateStore;

#[async_trait]
pub trait Module {
    fn register(
        &self,
        options: &mut FrameworkOptions<StateStore, anyhow::Error>,
        state_store: &StateStore,
    );

    fn configure_client(&self, builder: &mut ClientBuilder) {
        let _ = builder;
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

pub fn register_enabled(
    options: &mut FrameworkOptions<StateStore, anyhow::Error>,
    state_store: &StateStore,
) {
    let modules = inventory::iter::<&'static (dyn Module + Sync)>.into_iter();

    for module in modules.clone() {
        module.register(options, state_store);
    }

    info!("Registered {} enabled module(s)", modules.count());
}

pub fn configure_client(builder: &mut ClientBuilder) {
    let modules = inventory::iter::<&'static (dyn Module + Sync)>.into_iter();

    for module in modules {
        module.configure_client(builder);
    }

    info!("Applied client configuration for enabled module(s)");
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
