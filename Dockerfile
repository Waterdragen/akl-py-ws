# syntax=docker/dockerfile:1.2

FROM nginx

# Set working directory
COPY . /app
WORKDIR /app

# Install Go
RUN apt-get update && mkdir -p /root/go && \
    curl -L https://go.dev/dl/go1.22.0.linux-amd64.tar.gz --output /root/go/go1.22.0.linux-amd64.tar.gz && \
    tar -C /usr/local -xvf /root/go/go1.22.0.linux-amd64.tar.gz

# Install Python, pip, and python3-full
RUN apt-get update && apt-get install -y python3 python3-pip python3-full

# Install Rust
RUN apt-get update && curl https://sh.rustup.rs -sSf | sh -s -- -y

# Create and activate a python virtual environment
RUN python3 -m venv /venv
ENV PATH="/venv/bin:${PATH}"

# Add rust environment
ENV RUSTUP_HOME="/root/.rustup" \
    CARGO_HOME="/root/.cargo" \
    PATH="/root/.cargo/bin:${PATH}" \
    RUST_VERSION="1.75.0"
    
# Add go environment
ENV GOPATH="/root/go" \
    PATH="/usr/local/go/bin:${PATH}"
    
RUN cargo --version; \
    rustup --version; \
    rustc --version;
    
RUN go version

# Copy python dependencies
COPY python/requirements.txt .

# Disable nginx welcome page
RUN mv /etc/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf.disabled

# Copy nginx conf file
COPY nginx.conf /etc/nginx/conf.d/nginx.conf

# Copy templates to nginx directory
COPY templates /etc/nginx/templates

# Grant execute permissions to buildws.sh
RUN chmod +x /app/buildws.sh

# Install all dependencies
RUN sh /app/buildws.sh

# Grant execute permissions to startws.sh
RUN chmod +x /app/startws.sh

# Let Render detect service running on 8080
ENV PORT=8080

EXPOSE 8080

# Start websockets and nginx
CMD ["sh", "/app/startws.sh"]
