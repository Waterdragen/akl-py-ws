#!/bin/bash 
 
# Install python dependencies
pip3 install --no-cache-dir -r requirements.txt &&
 
# Install rust dependencies
cd ./rust &&
cargo clean &&
cargo build ;
