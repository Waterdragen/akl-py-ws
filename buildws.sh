#!/bin/bash 
 
# Install python dependencies
pip install -r python/requirements.txt &
 
# Install rust dependencies
cargo build --release --manifest-path rust/Cargo.toml 
