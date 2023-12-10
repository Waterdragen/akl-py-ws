#!/bin/bash 
 
# Start Django server for cmini
python3 python/manage.py runserver 1000 &
 
 
# Start Rocket server for oxeylyzer
cargo run --release --manifest-path rust/Cargo.toml 
