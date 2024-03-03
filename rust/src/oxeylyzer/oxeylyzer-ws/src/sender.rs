use std::fmt::{Debug, Formatter};
use std::sync::{Arc, Mutex};
use actix_ws::Session;
use tokio::runtime::Builder;

pub fn send_message<T>(session: Arc<Mutex<Session>>, message: T) where T: Into<String> {
    let message = message.into();
    Builder::new_current_thread()
        .enable_all()
        .build()
        .unwrap()
        .block_on(async {
            session.lock().unwrap().text(message).await.unwrap();
        });
}

pub trait Sendable {
    fn session(&self) -> &Arc<Mutex<Session>>;
    fn send<T>(&self, message: T) where T: Into<String> {
        let session = self.new_session();
        let message = message.into();

        (move || send_message(session, message))();
    }
    fn sendln<T>(&self, message: T) where T: Into<String> {
        let message = message.into() + "\n";
        self.send(message);
    }
    fn new_session(&self) -> Arc<Mutex<Session>> {
        Arc::clone(self.session())
    }
}

pub struct SessionWrapper {
    pub session: Arc<Mutex<Session>>,
}

impl Debug for SessionWrapper {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(f, "<Session object>")
    }
}

impl Clone for SessionWrapper {
    fn clone(&self) -> Self {
        SessionWrapper {session: Arc::clone(&self.session)}
    }
}

impl PartialEq for SessionWrapper {
    fn eq(&self, _other: &Self) -> bool {
        false
    }
}
