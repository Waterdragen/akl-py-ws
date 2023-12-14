# syntax=docker/dockerfile:1.2

FROM nginx

# Install Python, pip, and python3-full
RUN apt-get update && apt-get install -y python3 python3-pip python3-full

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

# Set working directory
COPY . /app
WORKDIR /app

# Create and activate a python virtual environment
RUN python3 -m venv /venv
ENV PATH="/venv/bin:$PATH"

# Add rust environment
ENV PATH="/root/.cargo/bin:$PATH"

# Copy python dependencies
COPY python/requirements.txt .

# Copy rust dependencies
COPY rust/Cargo.toml rust/Cargo.lock .

# Disable nginx welcome page
RUN mv /etc/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf.disabled

# Copy nginx conf file
COPY nginx.conf /etc/nginx/conf.d/nginx.conf

# Grant execute permissions to buildws.sh
RUN chmod +x /app/buildws.sh

# Grant execute permissions to startws.sh
RUN chmod +x /app/startws.sh

# Install all dependencies, start websockets and nginx
CMD bash /app/buildws.sh && bash /app/startws.sh && service nginx start
