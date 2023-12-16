use http::StatusCode;
use std::fs::File;
use std::io::{BufReader, Read};

use actix::{Actor, StreamHandler};
use actix_web::{web, App, Error, HttpResponse, HttpServer, HttpRequest, http};
use actix_web_actors::ws;

static SERVER_URL: &str = "127.0.0.1";
static SERVER_PORT: u32 = 9001;

// Define HTTP actor
struct MyWebSocket {
    // TODO: My WebSocket stats goes here
}

impl MyWebSocket {
    fn new() -> Self {
        // Create and return a new instance of the websocket state
        MyWebSocket {}
    }
}

impl Actor for MyWebSocket {
    type Context = ws::WebsocketContext<Self>;

    // TODO: Implement the WebSocket event handlers here
}

impl StreamHandler<Result<ws::Message, ws::ProtocolError>> for MyWebSocket {
    fn handle(&mut self, msg: Result<ws::Message, ws::ProtocolError>, context: &mut Self::Context) {
        match msg {
            Ok(ws::Message::Text(text)) => context.text(text),
            _ => (),
        }
    }
}

async fn ws_main(req: HttpRequest, stream: web::Payload) -> Result<HttpResponse, Error> {
    ws::start(MyWebSocket::new(), &req, stream)
}

async fn bad_request() -> HttpResponse {
    let html_file = File::open("../templates/bad_request.html").unwrap();
    let mut buf_reader = BufReader::new(html_file);
    let mut html = String::new();
    buf_reader.read_to_string(&mut html).unwrap();

    HttpResponse::build(StatusCode::BAD_REQUEST)
        .content_type("text/html; charset=utf-8")
        .body(html)
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    println!("Running server at https://{}:9001", SERVER_URL);
    HttpServer::new(|| {
        App::new()
            .service(web::resource("/rust").route(web::get().to(ws_main)))
            .default_service(web::route().to(bad_request))
    })
        .bind((SERVER_URL, SERVER_PORT))?
        .run()
        .await
}