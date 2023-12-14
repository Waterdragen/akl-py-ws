# syntax=docker/dockerfile:1.2

FROM nginx

# Install Python, pip, and python3-full
RUN apt-get update && apt-get install -y python3 python3-pip python3-full

# Set working directory
WORKDIR .

# Create and activate a virtual environment
RUN python3 -m venv /venv
ENV PATH="/venv/bin:$PATH"

# Install python dependencies
RUN pip3 install --no-cache-dir -r python/requirements.txt

# Disable nginx welcome page
RUN mv etc/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf.disabled

# Copy nginx conf file
COPY nginx.conf /etc/nginx/conf.d/nginx.conf

# Start websockets and nginx
CMD ./startws.sh && service nginx start
