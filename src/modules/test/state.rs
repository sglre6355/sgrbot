use tokio::sync::Mutex;

pub struct TestState {
    pub count: Mutex<u32>,
}
