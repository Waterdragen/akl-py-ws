use actix::{Actor, Addr, AsyncContext, Handler, StreamHandler};
use actix_web_actors::ws;
use actix_web_actors::ws::WebsocketContext;

use crate::messages::WsMessage;

pub struct OxeylyzerWs {
    addr: Option<Addr<Self>>,
    run: fn(Addr<Self>, String) -> Result<(), String>,
}

impl OxeylyzerWs {
    pub fn new(run: fn(addr: Addr<Self>, String) -> Result<(), String>) -> Self {
        OxeylyzerWs {addr: None, run }
    }

    fn process_command(&self,
                       command: String) -> () {
        let _session = (self.run)(self.addr.clone().unwrap(), command);
    }
}

// Event handler for websocket
impl Actor for OxeylyzerWs {
    type Context = WebsocketContext<Self>;

    fn started(&mut self, context: &mut Self::Context) {
        self.addr = Some(context.address().clone());
        println!("Rust websocket connection established");
    }

    fn stopped(&mut self, _: &mut Self::Context) {
        println!("Rust websocket connection closed");
    }
}

// Receive messages from client
impl StreamHandler<Result<ws::Message, ws::ProtocolError>> for OxeylyzerWs {
    fn handle(&mut self, msg: Result<ws::Message, ws::ProtocolError>, _context: &mut Self::Context) {
        match msg {
            Ok(ws::Message::Text(command)) => {
                // println!("Received command: {}", command);
                self.process_command(command.to_string());
            }
            _ => (),
        }
    }
}

impl Handler<WsMessage> for OxeylyzerWs {
    type Result = ();

    fn handle(&mut self, msg: WsMessage, ctx: &mut Self::Context) -> Self::Result {
        ctx.text(msg.0);
    }
}


