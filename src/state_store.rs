use std::{
    any::{Any, TypeId},
    sync::Arc,
};

use dashmap::DashMap;

#[derive(Debug, Default)]
pub struct StateStore {
    registry: DashMap<TypeId, Box<dyn Any + Send + Sync>>,
}

impl StateStore {
    pub fn insert<T: Any + Send + Sync>(&self, value: Arc<T>) {
        self.registry.insert(TypeId::of::<T>(), Box::new(value));
    }

    pub fn get<T: Any + Send + Sync>(&self) -> Option<Arc<T>> {
        self.registry
            .get(&TypeId::of::<T>())
            .and_then(|entry| entry.downcast_ref::<Arc<T>>().map(Arc::clone))
    }
}
