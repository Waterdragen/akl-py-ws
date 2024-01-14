#!/bin/bash 
 
# Install python dependencies
pip3 install --no-cache-dir -r requirements.txt &&
 
# Install rust dependencies
cargo build --release --manifest-path ./rust/Cargo.toml ;

directory="./rust/target/release"

if [ ! -d "$directory" ]; then
    echo "App deployment failed: wrong rust build directory"
    exit 1
fi
