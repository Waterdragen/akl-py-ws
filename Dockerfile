# syntax=docker/dockerfile:1.2

FROM nginx

# Install python and pip
RUN apt-get update && apt-get install -y python3 python3-pip

# Set working directory
WORKDIR /app
COPY . /app

# Install python dependencies
RUN pip3 install --no-cache-dir -r /app/python/requirements.txt

# Run python server
CMD ["python", "app/python/manage.py runserver"]

# Disable nginx welcome page
RUN mv /etc/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf.disabled

# Copy nginx conf file
COPY /app/nginx.conf /etc/nginx/conf.d/nginx.conf

# Start nginx and start websockets
CMD service nginx start && ./startws.sh
