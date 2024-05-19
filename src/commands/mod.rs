pub mod test;

pub type Context<'a> = poise::Context<'a, (), anyhow::Error>;
pub type Command = poise::Command<(), anyhow::Error>;

pub fn commands() -> Vec<Command> {
    [].into_iter().chain(test::commands()).collect()
}
