use http::StatusCode;
use std::fs::File;
use std::io::{BufReader, Read};
use std::panic::catch_unwind;
use std::sync::{Arc, Mutex};
use std::collections::HashMap;

use actix_web::{App, http, HttpRequest, HttpResponse, HttpServer, web};
use actix_ws::Message;
use futures::StreamExt;
use lazy_static::lazy_static;
use uuid::Uuid;

use oxeylyzer_repl::repl::{Repl, UserData};
use oxeylyzer_ws::sender::send_message;

static SERVER_URL: &str = "127.0.0.1";
static SERVER_PORT: u16 = 9001;

lazy_static! {
    static ref USERS_DATA: Arc<Mutex<HashMap<Uuid, Option<UserData>>>> = Arc::new(Mutex::new(HashMap::new()));
}

fn oxeylyzer_store_user_data(uuid: Uuid, user_data: Option<UserData>) {
    USERS_DATA.lock().unwrap().insert(uuid, user_data);
}

fn oxeylyzer_remove_user_data(uuid: Uuid) -> Option<UserData> {
    USERS_DATA.lock().unwrap().remove(&uuid).unwrap_or(None)
}

async fn oxeylyzer_ws_handler(req: HttpRequest, stream: web::Payload) -> Result<HttpResponse, actix_web::Error> {
    let (response, session, mut msg_stream) = actix_ws::handle(&req, stream)?;

    let session = Arc::new(Mutex::new(session));
    let uuid = Uuid::new_v4();

    actix_rt::spawn(async move {
        oxeylyzer_store_user_data(uuid, None);

        while let Some(Ok(msg)) = msg_stream.next().await {
            match msg {
                Message::Text(s) => {
                    // Process the client's message here

                    let s: String = s.to_string();
                    let session_ref = Arc::clone(&session);

                    tokio::task::spawn_blocking(move || {
                        let res = catch_unwind(|| {
                            let user_data = oxeylyzer_remove_user_data(uuid);

                            let repl_res = Repl::run(s, Arc::clone(&session_ref), user_data);
                            match repl_res {
                                Ok(repl) => {
                                    let user_data = repl.get_user_data();
                                    oxeylyzer_store_user_data(uuid, Some(user_data));
                                }
                                Err(err) => {
                                    send_message(Arc::clone(&session_ref), err);
                                }
                            }
                        });
                        if let Err(err) = res {
                            eprintln!("{:?}", err);
                            send_message(Arc::clone(&session_ref), format!("{:?}", err));
                        }
                    }).await.expect("failed to spawn blocking task");
                }
                _ => {}
            }
        }

        oxeylyzer_remove_user_data(uuid);
    });

    Ok(response)
}

async fn ws_main(req: HttpRequest, stream: web::Payload, path: web::Path<String>) -> Result<HttpResponse, actix_web::Error> {
    // Handle HTTP requests
    if !req.headers().contains_key("upgrade") {
        return Ok(bad_request().await.into());
    }

    // Handle websocket requests
    let tail = path.trim_end_matches("/");
    match tail {
        "oxeylyzer" => {
            oxeylyzer_ws_handler(req, stream).await
        },
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
    println!("Running server at http://{}:{}", SERVER_URL, SERVER_PORT);
    HttpServer::new(|| {
        App::new()
            .service(web::resource("/rust/{tail:.*}").route(web::get().to(ws_main)))
            .default_service(web::route().to(bad_request))
    })
        .bind((SERVER_URL, SERVER_PORT))?
        .run()
        .await
}