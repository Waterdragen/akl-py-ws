mod oxeylyzer;

use http::StatusCode;
use std::fs::File;
use std::io::{BufReader, Read};

use actix::{Actor, StreamHandler};
use actix_web::{App, Error, http, HttpRequest, HttpResponse, HttpServer, web};
use actix_web_actors::ws;

use oxeylyzer::Oxeylyzer;

static SERVER_URL: &str = "127.0.0.1";
static SERVER_PORT: u16 = 9001;

async fn ws_main(req: HttpRequest, stream: web::Payload, path: web::Path<String>) -> Result<HttpResponse, Error> {
    // Handle HTTP requests
    if !req.headers().contains_key("upgrade") {
        return Ok(bad_request().await.into());
    }

    // Handle websocket requests
    let tail = path.trim_end_matches("/");
    match tail {
        "oxeylyzer" => ws::start(Oxeylyzer::new(), &req, stream),
        _ => Ok(bad_request().await.into()),
    }

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
    println!("Running server at http://{}:9001", SERVER_URL);
    HttpServer::new(|| {
        App::new()
            .service(web::resource("/rust/{tail:.*}").route(web::get().to(ws_main)))
            .default_service(web::route().to(bad_request))
    })
        .bind((SERVER_URL, SERVER_PORT))?
        .run()
        .await
}