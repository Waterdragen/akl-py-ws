#!/bin/bash 
 
# Install python dependencies
pip3 install --no-cache-dir -r requirements.txt &&

# Install go dependencies
(cd ./go && go get ./... && go build) &&
 
# Install rust dependencies
(cd ./rust && cargo build --release)
