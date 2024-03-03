#!/bin/bash 

# Start Actix server for oxeylyzer
(cd ./rust && cargo run --release &) &&
 
# Start Django server for cmini, a200
python3 ./python/manage.py runserver 127.0.0.1:9000 &

# Start Gin server for genkey
(cd ./go && ./akl-ws) &
 
nginx -g 'daemon off;' &

echo "App is running" &

wait
