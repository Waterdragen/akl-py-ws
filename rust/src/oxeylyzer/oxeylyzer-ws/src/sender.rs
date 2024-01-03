use actix::{Actor, Addr, Handler, StreamHandler};
use actix_web_actors::ws;
use crate::messages::WsMessage;

pub trait Sendable<A>
    where A: Actor<Context = ws::WebsocketContext<A>> +
    StreamHandler<Result<ws::Message, ws::ProtocolError>> +
    Handler<WsMessage>,
{
    fn addr(&self) -> &Addr<A>;
    fn send(&self, msg: String) {
        self.addr().do_send(WsMessage(msg));
    }
    fn sendln(&self, msg: String) {
        self.addr().do_send(WsMessage(msg + "\n"));
    }
}
