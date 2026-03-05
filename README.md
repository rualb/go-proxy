# go-proxy

`go-proxy` is a microservice built with Go and the [Echo](https://echo.labstack.com/) web framework. It acts as a proxy service and includes features for TLS, automatic certificate management via Let's Encrypt (ACME), JWT authentication, Prometheus metrics, and GeoIP routing.

## Features

- **High-Performance HTTP Server:** Built on the Echo framework.
- **TLS & AutoTLS:** Supports standard TLS configuration as well as automatic TLS certificate provisioning using `golang.org/x/crypto/acme/autocert`.
- **Graceful Shutdown:** Ensures zero downtime during deployments or server restarts.
- **GeoIP Integration:** Supports IP geolocation using MaxMind DB (`geoip2-golang`).
- **Metrics:** Exposes Prometheus metrics for monitoring (`prometheus/client_golang`).
- **Security:** JWT-based authentication and TLS session caching.

## Prerequisites

- Go 1.26+
- Python 3.x (for running the build script `Makefile.py`)

## Configuration

The service uses JSON configuration files located in the `configs/go-proxy/` directory (e.g., `config.development.json`, `config.production.json`).

Example (`configs/go-proxy/config.development.json`):

```json
{
  "http_server": {
    "cert_dir": "./configs/cert",
    "cert_hosts": ["localhost"]
  }
}
```

The server dynamically configures its listening ports and TLS behaviors based on `listen`, `listen_tls`, and `auto_tls` flags in the configuration.

## Build and Run

The project uses a Python script (`Makefile.py`) to manage builds, tests, and execution.

### Run Tests

```sh
python Makefile.py test
```

### Build the Binary

You can compile the application for different operating systems (Windows, Linux, or Darwin).

```sh
# For Windows
python Makefile.py windows

# For Linux
python Makefile.py linux

# For macOS
python Makefile.py darwin
```

The resulting binary will be output to the `../../dist/` directory relative to the source structure, specifically as `dist/go-proxy`.

### Run Locally

To run the application with the default configurations:

```sh
python Makefile.py run
```

### Run Linter

```sh
python Makefile.py lint
```

## Architecture Context

`go-proxy` is designed as a microservice part of a larger architecture (see root `README.md`). It expects to run alongside other services like `go-auth`. For examples of the full application deployment using Docker Compose, refer to `projects/ecom-shop/README.md`.
