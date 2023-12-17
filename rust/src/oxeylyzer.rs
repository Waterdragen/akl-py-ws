use actix::{Actor, StreamHandler};
use actix_web_actors::ws;

use oxeylyzer_repl::repl;

pub struct Oxeylyzer {
}

impl Oxeylyzer {
    pub fn new() -> Self {
        // Create and return a new instance of the websocket state
        Oxeylyzer {}
    }

    pub fn run() -> Result<(), String> {
        repl::Repl::run()
    }
}

// Event handler for websocket
impl Actor for Oxeylyzer {
    type Context = ws::WebsocketContext<Self>;

    fn started(&mut self, context: &mut Self::Context) {
        println!("Rust websocket connection established");
    }

    fn stopped(&mut self, _: &mut Self::Context) {
        println!("Rust websocket connection closed");
    }
}

fn process_command(command: String) -> String {
    return format!("Response: {}", command);
}

// Receive messages from client
impl StreamHandler<Result<ws::Message, ws::ProtocolError>> for Oxeylyzer {
    fn handle(&mut self, msg: Result<ws::Message, ws::ProtocolError>, context: &mut Self::Context) {
        match msg {
            Ok(ws::Message::Text(command)) => {
                println!("Received command: {}", command);
                let response = process_command(command.to_string());

                // Send response back to client
                context.text(response)
            }
            _ => (),
        }
    }
}

