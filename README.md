> **Work in Progress** — This project is under active development and is not yet ready for production use. APIs, configuration, and features may change without notice. Star the repo and check back soon!

# OXI Metric

**Open-source, self-hosted, privacy-first analytics for mobile and desktop apps. Free for commercial use.**

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![CI](https://github.com/oxisoft/oximetric/actions/workflows/ci.yml/badge.svg)](https://github.com/oxisoft/oximetric/actions/workflows/ci.yml)

---

## What is OXI Metric?

OXI Metric is an **open-source, on-premises analytics platform** that gives you complete control over your data. Track events, user behavior, and app performance across iOS, Android, macOS, Windows, Linux, and Web -- without sending a single byte to third-party servers.

Unlike cloud analytics services like Google Analytics, Mixpanel, or Amplitude, OXI Metric runs entirely on **your own infrastructure**. Your data never leaves your servers. No vendor lock-in. No surprise pricing. No compliance headaches.

Built with Go for the server and a [pure Dart Flutter SDK](https://github.com/oxisoft/oximetric-flutter-sdk) for client-side integration, OXI Metric is designed to be lightweight, fast, and easy to deploy with a single `docker compose up`. It is completely **free for commercial use** under the MIT license.

---

## Key Features

- **Self-hosted & on-premises** -- deploy on your own servers, your cloud, or air-gapped environments
- **Privacy by design** -- no raw PII stored, anonymous device and user IDs, server-side geolocation only
- **Multi-platform SDK** -- [Flutter SDK](https://github.com/oxisoft/oximetric-flutter-sdk) supports iOS, Android, macOS, Windows, Linux, and Web
- **Rich event tracking** -- capture events with multiple typed key-value properties (string, int, float, boolean, datetime) in a single call
- **Built-in analytics console** -- Bootstrap 5 dashboard with charts, event explorer, geographic distribution, device analytics
- **Dual database support** -- SQLite for small deployments, PostgreSQL for scale
- **Role-based access control** -- admin, manager, and viewer roles with granular permissions
- **Two-factor authentication** -- TOTP-based 2FA compatible with Google Authenticator, Authy, and other authenticator apps
- **Multiple project tokens** -- manage separate tokens per app version, disable old tokens without affecting new ones
- **Server-side IP geolocation** -- country and city detection using DB-IP Lite, no client-side location permissions needed
- **Docker-ready** -- production-ready Docker images with Watchtower support for automatic updates
- **Security hardened** -- bcrypt passwords, JWT with short expiry, rate limiting, security headers, timing attack protection, non-root Docker container

---

## Why Self-Hosted Analytics?

Cloud analytics platforms process your users' data on servers you don't control, in jurisdictions you may not have chosen, under terms of service that can change at any time.

**With OXI Metric, you get:**

- **Complete data ownership** -- your analytics data lives on your servers, in your databases, under your control
- **Regulatory compliance** -- simplify GDPR, HIPAA, CCPA, and other privacy regulation compliance by keeping data within your infrastructure boundaries
- **No data sharing** -- your users' behavioral data is never shared with ad networks, data brokers, or third parties
- **100% free for commercial use** -- MIT licensed, no per-event pricing, no usage tiers, no surprise bills. Run on a $5/month VPS or a multi-server cluster
- **Network independence** -- works in air-gapped environments, private networks, and restricted corporate infrastructures
- **Auditability** -- full access to source code, database, and logs for security audits and compliance reviews

---

## Privacy & Compliance

OXI Metric is built for teams that take user privacy seriously:

- **No raw personal data** -- user identifiers are SHA-256 hashed before leaving the device. The server never sees usernames, emails, or real user IDs
- **Anonymous device tracking** -- each device gets a random UUID, not a hardware fingerprint
- **Server-side geolocation** -- country and city are resolved from IP addresses on the server using DB-IP Lite. No GPS, no location permissions, no precise coordinates
- **IP address handling** -- IP addresses are stored for geolocation only and can be purged with configurable data retention policies
- **Data retention controls** -- set per-project retention periods to automatically purge old data
- **Opt-out support** -- the SDK provides a simple API for users to disable analytics (e.g., when declining tracking in app settings)
- **Open source** -- inspect every line of code that handles your data. No black boxes, no hidden telemetry

OXI Metric helps you comply with **GDPR**, **CCPA**, **PIPEDA**, **LGPD**, and other data protection regulations by minimizing data collection and keeping everything on your infrastructure.

---

## Quick Start

Get OXI Metric running in 30 seconds:

```bash
# Create environment file
cat > .env << 'EOF'
OXIMETRIC_ADMIN_USERNAME=admin
OXIMETRIC_ADMIN_PASSWORD=your-secure-password
OXIMETRIC_JWT_SECRET=change-this-to-a-random-string-at-least-32-chars
EOF

# Download and start
curl -O https://raw.githubusercontent.com/oxisoft/oximetric/main/deploy/docker-compose.sqlite.yml
docker compose -f docker-compose.sqlite.yml up -d
```

Open **http://localhost:6940** and log in with your admin credentials.

---

## Installation

### Docker Compose with SQLite

Best for small to medium deployments. Zero external dependencies.

```bash
cd deploy/
cp .env.example .env
# Edit .env with your credentials

docker compose -f docker-compose.sqlite.yml up -d
```

### Docker Compose with PostgreSQL

Best for larger deployments with high event volumes.

```bash
cd deploy/
cp .env.example .env
# Edit .env — also set OXIMETRIC_DB_PASSWORD

docker compose -f docker-compose.postgres.yml up -d
```

### Docker Compose with Watchtower (Auto-Updates)

Automatically pull and deploy the latest OXI Metric version daily:

```bash
# SQLite with auto-updates
docker compose -f docker-compose.sqlite.watchtower.yml up -d

# PostgreSQL with auto-updates
docker compose -f docker-compose.postgres.watchtower.yml up -d
```

Watchtower checks for new images every 24 hours and restarts the container with the latest version.

### Build from Source

```bash
git clone https://github.com/oxisoft/oximetric.git
cd oximetric
make build
```

The binary embeds the web console -- no separate frontend deployment needed.

---

## SQLite vs PostgreSQL

OXI Metric supports both SQLite and PostgreSQL. Choose based on your scale:

| | SQLite | PostgreSQL |
|---|---|---|
| **Setup** | Zero config, single file | Separate database server |
| **Deployment** | Single container | Two containers |
| **Concurrent writes** | Sequential (WAL mode) | Fully concurrent |
| **Backup** | Copy a single file | pg_dump / replication |
| **Best for** | Development, small apps | Production, high traffic |

### Practical Recommendations

| Daily Active Users | Recommendation |
|---|---|
| **< 10,000 DAU** | **SQLite** -- more than enough. Simpler to operate, back up, and migrate. Handles thousands of events per second. |
| **10,000 - 100,000 DAU** | **Either works** -- SQLite handles this well with WAL mode, but PostgreSQL gives you headroom for growth. |
| **> 100,000 DAU** | **PostgreSQL** -- concurrent write handling and connection pooling become important at scale. |
| **> 1,000,000 DAU** | **PostgreSQL** with tuned connection pool, dedicated hardware, and consider read replicas. |

You can start with SQLite and migrate to PostgreSQL later -- the schema is compatible and data can be exported/imported.

---

## Configuration

All configuration is done through environment variables:

| Variable | Required | Default | Description |
|---|---|---|---|
| `OXIMETRIC_ADMIN_USERNAME` | Yes | -- | Default admin username (created/updated on every start) |
| `OXIMETRIC_ADMIN_PASSWORD` | Yes | -- | Default admin password (min 8 characters) |
| `OXIMETRIC_JWT_SECRET` | Yes | -- | JWT signing secret (min 32 characters) |
| `OXIMETRIC_DB_DRIVER` | No | `sqlite` | Database driver: `sqlite` or `postgres` |
| `OXIMETRIC_DB_DSN` | No | `./oximetric.db` | Database connection string |
| `OXIMETRIC_PORT` | No | `6940` | HTTP server port |
| `OXIMETRIC_GEOIP_DB_PATH` | No | `./data/dbip-city-lite.mmdb` | Path to GeoIP database |
| `OXIMETRIC_LOG_LEVEL` | No | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `OXIMETRIC_DOMAIN_NAME` | No | -- | Public URL of your OXI Metric instance (e.g. `https://analytics.yourcompany.com`). Used in Help page code examples. |
| `OXIMETRIC_TRUSTED_PROXIES` | No | -- | Comma-separated IPs of trusted reverse proxies (empty = trust all) |

---

## API Overview

OXI Metric exposes a RESTful API for event ingestion, management, and analytics.

### Tracking API (SDK to Server)

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/track` | Ingest batch of events |
| `POST` | `/api/v1/device` | Register or update device |
| `POST` | `/api/v1/identify` | Link device to anonymous user |

### Management API

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Login (username or email) |
| `GET` | `/api/v1/projects` | List projects |
| `POST` | `/api/v1/projects` | Create project |
| `POST` | `/api/v1/projects/:id/tokens` | Create tracking token |
| `GET` | `/api/v1/users` | List console users |

### Analytics API

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/analytics/:id/overview` | Dashboard overview |
| `GET` | `/api/v1/analytics/:id/events` | Event counts and time series |
| `GET` | `/api/v1/analytics/:id/devices` | Device/platform distribution |
| `GET` | `/api/v1/analytics/:id/geo` | Geographic distribution |
| `GET` | `/api/v1/analytics/:id/users` | User analytics |

All tracking endpoints authenticate via `X-Token` header. Console endpoints use `Authorization: Bearer <jwt>`.

For the full API specification, see [REQUIREMENTS.md](../REQUIREMENTS.md).

---

## Analytics Console

OXI Metric includes a built-in web console embedded directly into the server binary -- no separate frontend deployment required.

- **Dashboard** -- key metrics, event time series, top events, geographic summary
- **Event Explorer** -- browse events by name, drill into property breakdowns
- **Device Analytics** -- platform and OS distribution charts
- **Geographic View** -- country and city breakdown tables
- **User Analytics** -- unique users over time, retention data
- **Project Management** -- create projects, manage tracking tokens with labels
- **User Management** -- create console users with admin/manager/viewer roles
- **Account Settings** -- password change, two-factor authentication setup

---

## Flutter SDK

Integrate OXI Metric into your Flutter app with the official SDK:

**[oximetric-flutter-sdk](https://github.com/oxisoft/oximetric-flutter-sdk)**

The SDK is pure Dart with zero native dependencies, supporting all platforms: iOS, Android, macOS, Windows, Linux, and Web.

```dart
// Initialize
await OxiMetric.initialize(
  serverUrl: 'https://analytics.yourcompany.com',
  token: 'your-project-token',
);

// Track events with typed properties
OxiMetric.track('purchase', properties: {
  'amount': 29.99,
  'currency': 'USD',
  'is_first_purchase': true,
});

// Identify user (anonymous hash)
OxiMetric.identify('user@example.com');
```

---

## Development

```bash
# Run locally with SQLite
make dev

# Populate with sample data
OXIMETRIC_TOKEN=<your-token> make seed

# Run unit tests
make test-unit

# Run integration tests (SQLite + PostgreSQL in Docker)
make test-integration

# Run all tests with coverage report
make test-coverage

# Build binary
make build

# Build Docker image
make docker-build

# Show all available commands
make help
```

---

## Security

OXI Metric is built with security as a core concern:

- **Password hashing** -- bcrypt with configurable cost factor
- **JWT authentication** -- 4-hour token expiry, HMAC-SHA256 signing, minimum 32-character secret
- **Two-factor authentication** -- TOTP-based 2FA with brute force protection (rate-limited to 5 attempts per 5 minutes)
- **Rate limiting** -- 1000 req/min per tracking token, 10 req/min on login endpoint
- **Security headers** -- X-Frame-Options, Content-Security-Policy, X-Content-Type-Options, Referrer-Policy
- **Input validation** -- request body size limits (1MB), event batch limits (100), property limits (50), password minimum length (8 characters)
- **Timing attack protection** -- constant-time login checks prevent username enumeration
- **XSS prevention** -- all dynamic content escaped before DOM insertion
- **SQL injection protection** -- parameterized queries exclusively, no string concatenation
- **Trusted proxy support** -- configurable X-Forwarded-For trust list
- **Non-root Docker** -- container runs as unprivileged user
- **Audit logging** -- all authentication and management operations logged with user ID and IP

---

## Contributing

Contributions are welcome! Here's how to get started:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes and add tests
4. Run the test suite: `make test-all`
5. Ensure coverage stays above 80%
6. Submit a pull request

Please open an issue first for major changes to discuss the approach.

---

## License

OXI Metric is released under the [MIT License](LICENSE) -- **free for commercial use** with no restrictions. Use it in your startup, enterprise, SaaS product, or any commercial project at no cost.

---

## Links

- **Website**: [oxisoft.io](https://oxisoft.io)
- **GitHub**: [github.com/oxisoft/oximetric](https://github.com/oxisoft/oximetric)
- **Flutter SDK**: [github.com/oxisoft/oximetric-flutter-sdk](https://github.com/oxisoft/oximetric-flutter-sdk)
- **X**: [@oxisoftio](https://x.com/oxisoftio)

---

<p align="center">
  Built with care by <a href="https://oxisoft.io">OxiSoft</a><br>
  Copyright 2026 OxiSoft. All rights reserved.
</p>
