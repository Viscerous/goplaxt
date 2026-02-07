# Plaxt

![Build Status](https://github.com/Viscerous/goplaxt/actions/workflows/build.yaml/badge.svg)

**Plaxt** is a lightweight, self-hosted service that synchronises your Plex viewing history to Trakt.tv. It acts as a bridge, receiving webhooks from Plex and scrobbling your plays, rates, and collections to Trakt automatically.

---

## Features

- **Self-Hosted**: Run it locally alongside your Plex server using Docker.
- **Secure Authentication**: Uses OAuth 2.0 Device Flow, no callback URLs required.
- **Lightweight**: Written in Go for minimal resource usage and high performance.
- **Multi-User Support**: Supports multiple users on a single instance using a single Trakt API application.
- **Easy Integration**: Works with standard Plex Webhooks (requires Plex Pass, but not Trakt VIP).

## Getting Started

### 1. Create a Trakt Application

Before running Plaxt, you need an API key from Trakt.

1. Go to [Trakt API Applications](https://trakt.tv/oauth/applications) and create a new application.
2. Set **Redirect uri** to: `urn:ietf:wg:oauth:2.0:oob`
3. Enable **/checkin** and **/scrobble** permissions.
4. Save the application and copy your **Client ID** and **Client Secret**.

### 2. Run with Docker

You can spin up Plaxt in seconds.

#### Docker CLI

```bash
docker create \
  --name=plaxt \
  --restart always \
  -v ./keystore:/app/keystore \
  -e TRAKT_ID="<CLIENT_ID>" \
  -e TRAKT_SECRET="<CLIENT_SECRET>" \
  -p 8000:8000 \
  ghcr.io/viscerous/goplaxt:latest

docker start plaxt
```

#### Docker Compose

```yaml
services:
  plaxt:
    container_name: plaxt
    image: ghcr.io/viscerous/goplaxt:latest
    restart: unless-stopped
    ports:
      - 8000:8000
    environment:
      - TRAKT_ID=<CLIENT_ID>
      - TRAKT_SECRET=<CLIENT_SECRET>
    volumes:
      - ./keystore:/app/keystore
```

### 3. Setup & Authenticate

1. Open your browser and navigate to your server's address: `http://<YOUR_SERVER_IP>:8000`.
   > **Note**: Do not use `localhost` if you are setting this up for a remote server or container; use the actual LAN IP (e.g., `192.168.1.50`).
2. Click **Connect with Trakt** and follow the instructions to link your account.
3. Once authenticated, the dashboard will display a **Webhook URL**.
4. Copy this URL and add it to your [Plex Webhooks Settings](https://app.plex.tv/desktop/#!/settings/webhooks).

### 4. Multiple Users

Plaxt supports an unlimited number of users on a single instance. 
- Only **one** Trakt API application needs to be created by the administrator.
- Each user visits the dashboard and completes the setup independently.
- Every user receives their own unique Webhook URL, allowing Plaxt to route events to the correct Trakt account.

## Configuration

Plaxt is configured primarily via environment variables.

| Variable | Description | Required | Default |
|----------|-------------|:--------:|:-------:|
| `TRAKT_ID` | Your Trakt Application Client ID | ✅ | - |
| `TRAKT_SECRET` | Your Trakt Application Client Secret | ✅ | - |
| `ALLOWED_HOSTNAMES` | Permitted hostnames for the web UI (security) | ❌ | - |
| `LISTEN` | Address/Port to listen on | ❌ | `0.0.0.0:8000` |
| `POSTGRESQL_URL`| Connection string for PostgreSQL (optional) | ❌ | - |
| `REDIS_URI` | Connection string for Redis (optional) | ❌ | - |
| `JSON_LOGS` | Enable structured JSON logging | ❌ | `false` |
| `LOG_LEVEL` | Logging verbosity (DEBUG, INFO, WARN, ERROR) | ❌ | `INFO` |

> *Note: By default, Plaxt uses a simple on-disk store mounted at `/app/keystore`. Redis or PostgreSQL are optional alternatives for stateless deployments.*

## Contributing

This project is a modern fork of the original `goplaxt` by XanderStrike.

## License

MIT
