#!/bin/bash 
 
# Install python dependencies
pip3 install --no-cache-dir -r requirements.txt &&
 
# Install rust dependencies
source /opt/render/project/.cargo/env &&
cargo build --release --manifest-path ./rust/Cargo.toml ;
