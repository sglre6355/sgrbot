use thiserror::Error;

#[derive(Debug, Error)]
pub enum ModuleError {
    #[error("state for `{module_name}` module is not registered")]
    StateNotRegistered { module_name: String },
}
