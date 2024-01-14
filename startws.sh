#!/bin/bash 

# Start Actix server for oxeylyzer
 
# Start Django server for cmini
python3 ./python/manage.py runserver 127.0.0.1:9000 &
 
nginx -g 'daemon off;' &

echo "App is running" &

wait
