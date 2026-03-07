# go-proxy

A high-performance reverse proxy server written in Go with built-in security features, geo-blocking, rate limiting, and TLS support.

## Features

- 🚀 **High Performance** - Built on Echo framework with configurable HTTP transport
- 🔒 **Security** - CSRF protection, rate limiting, content security policy
- 🌍 **Geo-blocking** - Country-based access control using GeoIP2
- 🔄 **Load Balancing** - Round-robin distribution across multiple upstreams
- 🔐 **TLS Support** - Manual and automatic (Let's Encrypt) certificate management
- 📊 **Monitoring** - Built-in Prometheus metrics endpoint
- ⚙️ **Flexible Configuration** - Support for JSON configs, environment variables, and CLI flags
- 🔁 **URL Rewriting** - Path rewrite rules for upstream requests
- 🛡️ **Maintenance Mode** - Built-in maintenance page support

## Quick Start

### Installation
```bash
# Clone the repository
git clone https://github.com/yourusername/go-proxy.git
cd go-proxy

# Build
go build -o go-proxy ./cmd/go-proxy

# Run
./go-proxy -config ./configs -listen 127.0.0.1:8080
```

### Docker
```bash
docker run -d \
  -p 80:80 \
  -v /path/to/configs:/app/configs:ro \
  -e APP_LISTEN=0.0.0.0:80 \
  -e APP_UPSTREAMS='["http://backend1:8080","http://backend2:8080"]' \
  go-proxy
```

## Configuration

### Command Line Flags
```bash
./go-proxy \
  -config ./configs \
  -listen 127.0.0.1:80 \
  -listen-tls 127.0.0.1:443 \
  -cert-dir ./certs \
  -cert_host example.com \
  -upstream http://backend:8080/api \
  -geo-ip-file ./GeoLite2-Country.mmdb
```

### Environment Variables

All configuration can be set via environment variables with `APP_` prefix:
```bash
APP_ENV=production
APP_LISTEN=0.0.0.0:80
APP_LISTEN_TLS=0.0.0.0:443
APP_HTTP_ACCESS_LOG=true
APP_HTTP_RATE_LIMIT=10
APP_HTTP_RATE_BURST=20
APP_CERT_DIR=/app/cert
APP_CERT_HOSTS='["example.com","www.example.com"]'
APP_UPSTREAMS='["http://backend1:8080","http://backend2:8080"]'
APP_GEO_IP_FILE=/app/geo-ip/GeoLite2-Country.mmdb
APP_ALLOW_COUNTRY='["US","CA","GB"]'
APP_BLOCK_COUNTRY='["XX","YY"]'
```

### JSON Configuration

Create `config.production.json` in your config directory:
```json
{
  "env": "production",
  "title": "My Proxy",
  "is_maint": false,
  "http_server": {
    "listen": "0.0.0.0:80",
    "listen_tls": "0.0.0.0:443",
    "auto_tls": true,
    "cert_dir": "/app/cert",
    "cert_hosts": ["example.com", "www.example.com"],
    "redirect_https": true,
    "redirect_www": true,
    "access_log": true,
    "rate_limit": 10,
    "rate_burst": 20,
    "read_timeout": 5,
    "write_timeout": 10,
    "idle_timeout": 30,
    "request_timeout": 20,
    "body_limit": "2M",
    "csrf": true,
    "tls_session_cache": true,
    "tls_session_tickets": true,
    "allow_origins": ["https://example.com"],
    "headers_del": ["Server", "X-Powered-By"],
    "headers_add": ["X-Frame-Options: SAMEORIGIN"],
    "content_policy": "default-src 'self'"
  },
  "proxy": {
    "upstreams": [
      "http://backend1:8080/api",
      "http://backend2:8080/api"
    ],
    "override_status": {
      "502": "/502.html",
      "503": "/maintenance.html"
    }
  },
  "geo_ip": {
    "enabled": true,
    "file": "/app/geo-ip/GeoLite2-Country.mmdb",
    "allow_country": ["US", "CA", "GB"],
    "block_country": []
  },
  "http_transport": {
    "max_idle_conns": 100,
    "max_idle_conns_per_host": 10,
    "idle_conn_timeout": 90,
    "max_conns_per_host": 50
  }
}
```

## Advanced Usage

### Multiple Upstreams with Load Balancing
```bash
# Multiple servers for the same path (round-robin)
./go-proxy \
  -upstream "http://server1:8080/api?server=server2:8080&server=server3:8080"
```

### URL Rewriting
```bash
# Rewrite /old to /new on upstream
./go-proxy \
  -upstream "http://backend:8080/api?rewrite=/old:/new&rewrite=/v1:/v2"
```

### GeoIP Blocking

Download GeoLite2 database:
```bash
# Get GeoLite2-Country.mmdb from MaxMind
wget https://example.com/GeoLite2-Country.mmdb

./go-proxy \
  -geo-ip-file ./GeoLite2-Country.mmdb \
  -upstream http://backend:8080
```

Set country restrictions via environment:
```bash
APP_GEO_IP_ENABLED=true
APP_ALLOW_COUNTRY='["US","CA","GB"]'  # Whitelist
# OR
APP_BLOCK_COUNTRY='["CN","RU"]'       # Blacklist
```

### TLS Configuration

#### Manual Certificates
```bash
./go-proxy \
  -listen-tls 0.0.0.0:443 \
  -cert-dir /path/to/certs \
  -cert_host example.com
```

Expected file structure:
```
/path/to/certs/
  └── example.com/
      ├── cert.pem
      └── key.pem
```

#### Automatic TLS (Let's Encrypt)
```json
{
  "http_server": {
    "listen_tls": "0.0.0.0:443",
    "auto_tls": true,
    "cert_dir": "/app/cert",
    "cert_hosts": ["example.com", "www.example.com"]
  }
}
```

### Monitoring

Access Prometheus metrics:
```bash
# Configure metrics endpoint
APP_HTTP_SYS_METRICS=true
APP_HTTP_LISTEN_SYS=127.0.0.1:9090
APP_HTTP_SYS_API_KEY=your-secret-key

# Access metrics
curl http://127.0.0.1:9090/sys/api/metrics?api-key=your-secret-key
```

### Custom Error Pages

Place HTML files in `web/pages/`:
```
web/pages/
  ├── 502.html
  ├── 503.html
  └── maint.html
```

Configure redirects:
```json
{
  "proxy": {
    "override_status": {
      "502": "/502.html",
      "503": "/maintenance.html"
    }
  }
}
```

### Maintenance Mode
```bash
# Enable via flag
./go-proxy -is-maint

# Or via environment
APP_IS_MAINT=true ./go-proxy
```

## Architecture
```
                                    ┌──────────────┐
                                    │   GeoIP DB   │
                                    └──────┬───────┘
                                           │
┌─────────┐      ┌────────────────────────▼────────┐
│ Client  │─────▶│         go-proxy                │
└─────────┘      │  ┌──────────────────────────┐   │
                 │  │ Middleware Stack         │   │
                 │  │ - Recovery               │   │
                 │  │ - GeoIP Blocking         │   │
                 │  │ - Logger                 │   │
                 │  │ - Maintenance Mode       │   │
                 │  │ - HTTPS/WWW Redirect     │   │
                 │  │ - Content Security       │   │
                 │  │ - Rate Limiting          │   │
                 │  │ - CSRF Protection        │   │
                 │  │ - Reverse Proxy          │   │
                 │  └──────────┬───────────────┘   │
                 └─────────────┼───────────────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
              ▼                ▼                ▼
      ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
      │  Backend 1   │ │  Backend 2   │ │  Backend 3   │
      └──────────────┘ └──────────────┘ └──────────────┘
```

## Environment Files

Use `_FILE` suffix to load secrets from files:
```bash
APP_DB_PASSWORD_FILE=/run/secrets/db_password
APP_HTTP_SYS_API_KEY_FILE=/run/secrets/api_key
```

## Health Checks

Built-in health endpoints:
```bash
# Public health check
curl http://localhost/health

# Proxy ping (requires CSRF token for POST)
curl http://localhost/proxy/api/ping
curl http://localhost/proxy/api/status
```

## Performance Tuning

### HTTP Transport
```json
{
  "http_transport": {
    "max_idle_conns": 100,
    "max_idle_conns_per_host": 10,
    "idle_conn_timeout": 90,
    "max_conns_per_host": 50
  }
}
```

### Server Timeouts
```json
{
  "http_server": {
    "read_timeout": 5,
    "write_timeout": 10,
    "idle_timeout": 30,
    "request_timeout": 20
  }
}
```

### TLS Optimization
```json
{
  "http_server": {
    "tls_session_cache": true,
    "tls_session_cache_size": 128,
    "tls_session_tickets": true
  }
}
```

## Docker Compose Example
```yaml
version: '3.8'

services:
  go-proxy:
    image: alpine:3.20
    container_name: go-proxy
    restart: always
    command: ./go-proxy
    user: 30100:30100
    ports:
      - "80:80"
      - "443:443"
    environment:
      - APP_CONFIG=/app/configs
      - APP_LISTEN=0.0.0.0:80
      - APP_LISTEN_TLS=0.0.0.0:443
      - APP_CERT_DIR=/app/cert
      - APP_CERT_HOSTS=["example.com"]
      - APP_GEO_IP_FILE=/app/geo-ip/GeoLite2-Country.mmdb
      - APP_DB_PASSWORD_FILE=/app/secret/__db__
    volumes:
      - ./configs:/app/configs:ro
      - ./certs:/app/cert:ro
      - ./geo-ip:/app/geo-ip:ro
      - ./secret:/app/secret:ro
      - ./go-proxy:/app/go-proxy:ro
      - /etc/ssl/certs:/etc/ssl/certs:ro
    working_dir: /app/go-proxy
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "5"
    networks:
      - backend

networks:
  backend:
    driver: bridge
```

## Building

### Standard Build
```bash
go build -o go-proxy ./cmd/go-proxy
```

### With Version Info
```bash
VERSION=1.0.0
COMMIT=$(git rev-parse HEAD)
SHORT_COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build \
  -ldflags "-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.ShortCommit=$SHORT_COMMIT -X main.Date=$DATE" \
  -o go-proxy \
  ./cmd/go-proxy

./go-proxy -version
```

### Docker Build
```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o go-proxy ./cmd/go-proxy

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/go-proxy /app/go-proxy
USER 30100:30100
WORKDIR /app
CMD ["./go-proxy"]
```

## Testing
```bash
# Run all tests
go test ./...

# Run E2E tests
go test ./test/e2e/...

# Run with coverage
go test -cover ./...
```

## Security Considerations

1. **Run as non-root user** - Use unprivileged UID/GID (e.g., 30100:30100)
2. **Read-only volumes** - Mount configs and binaries as read-only
3. **Secret management** - Use `_FILE` suffix for sensitive data
4. **CSRF protection** - Enabled by default, disable only if necessary
5. **Rate limiting** - Configure appropriate limits for your use case
6. **TLS certificates** - Use Let's Encrypt or valid certificates
7. **GeoIP blocking** - Restrict access by country if needed
8. **System metrics** - Protect with API key authentication

## Troubleshooting

### Port Permission Denied
```bash
# On Linux, ports < 1024 require root or capabilities
sudo setcap 'cap_net_bind_service=+ep' ./go-proxy

# Or use ports >= 1024
./go-proxy -listen 0.0.0.0:8080
```

### Certificate Issues
```bash
# Check certificate files exist
ls -la /path/to/certs/example.com/

# Expected files: cert.pem, key.pem
```


## License

This project is licensed under the MIT License - see the LICENSE file for details.
