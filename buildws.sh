#!/bin/bash 
 
# Install python dependencies
pip3 install --no-cache-dir -r requirements.txt &
 
# Install rust dependencies
cargo build --release --manifest-path Cargo.toml 
