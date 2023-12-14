#!/bin/bash 
 
# Install python dependencies
pip3 install -r python/requirements.txt &
 
# Install rust dependencies
cargo build --release --manifest-path rust/Cargo.toml 
