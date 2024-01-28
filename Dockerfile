# syntax=docker/dockerfile:1.2


FROM nginx

# Set working directory
WORKDIR /app
COPY . .

# Disable nginx welcome page
RUN mv /etc/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf.disabled

# Copy nginx conf file
COPY nginx.conf /etc/nginx/conf.d/nginx.conf

# Copy templates to nginx directory
COPY templates /etc/nginx/templates


FROM rust:1.67

# Set working directory
WORKDIR /app
COPY . .

# Add rust environment
ENV RUSTUP_HOME="/opt/render/project/.rustup" \
    CARGO_HOME="/opt/render/project/.cargo" \
    PATH="/opt/render/project/.cargo/bin:${PATH}" \
    RUST_VERSION="1.75.0"
    
RUN cargo build --release --manifest-path ./rust/Cargo.toml ;

FROM python:3.11

# Set working directory
WORKDIR /app
COPY . .

# Create and activate a python virtual environment
RUN python3 -m venv /venv
ENV PATH="/venv/bin:$PATH"

# Copy python dependencies
COPY python/requirements.txt .

# Install python dependencies
RUN pip3 install --no-cache-dir -r requirements.txt

# Grant execute permissions to startws.sh
RUN chmod +x /app/startws.sh

# Let Render detect service running on 8080
ENV PORT=8080

EXPOSE 8080

# Start websockets and nginx
CMD ["sh", "/app/startws.sh"]
