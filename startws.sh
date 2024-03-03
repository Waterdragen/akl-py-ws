#!/bin/bash 

# Start Actix server for oxeylyzer
(cd ./rust && cargo run --release &) &&
 
# Start Django server for cmini
python3 ./python/manage.py runserver 127.0.0.1:9000 &

# Test Go
(cd ./go && go run main.go) &
 
nginx -g 'daemon off;' &

echo "App is running" &

wait
