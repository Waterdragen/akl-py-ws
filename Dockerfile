# syntax=docker/dockerfile:1.2

FROM nginx

# Install Python, pip, and python3-full
RUN apt-get update && apt-get install -y python3 python3-pip python3-full

# Install Rust
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y

# Set working directory
COPY . /app
WORKDIR /app

# Create and activate a python virtual environment
RUN python3 -m venv /venv
ENV PATH="/venv/bin:$PATH"

# Add rust environment
ENV RUSTUP_HOME="/opt/render/project/.rustup" \
    CARGO_HOME="/opt/render/project/.cargo" \
    PATH="/opt/render/project/.cargo/bin:${PATH}" \
    RUST_VERSION="1.75.0"

# Copy python dependencies
COPY python/requirements.txt .

# Disable nginx welcome page
RUN mv /etc/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf.disabled

# Copy nginx conf file
COPY nginx.conf /etc/nginx/conf.d/nginx.conf

# Copy templates to nginx directory
COPY templates /etc/nginx/templates

# Grant execute permissions to buildws.sh
# RUN chmod +x /app/buildws.sh

# Install all dependencies
# RUN /app/buildws.sh

RUN rustup install stable
RUN rustup default stable
RUN exec $SHELL
RUN pip3 install --no-cache-dir -r requirements.txt
RUN rustup run stable cargo build --release --manifest-path ./rust/Cargo.toml


# Grant execute permissions to startws.sh
RUN chmod +x /app/startws.sh

# Let Render detect service running on 8080
ENV PORT=8080

EXPOSE 8080

# Start websockets and nginx
CMD ["sh", "/app/startws.sh"]
