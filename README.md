# go-twitch-channel-point-miner

Channel point miner for Twitch written in Go.

## Usage

### Using Docker

Download the `tcpm.example.yaml` and rename it to `tcpm.yaml`:

```bash
wget https://raw.githubusercontent.com/le0developer/go-twitch-channel-point-miner/master/tcpm.example.yaml -O tcpm.yaml
```

Adjust the configuration to your needs.

Create a `docker-compose.yml` file:

```yaml
services:
  tcpm:
    image: ghcr.io/le0developer/go-twitch-channel-point-miner:latest
    restart: unless-stopped
    command: ["run", "--login"]
    volumes:
      - ./tcpm.yaml:/tcpm.yaml
      # The container does not have its own certificate store, so you'll need to mount the host's certificate store
      - /etc/ssl/certs:/etc/ssl/certs:ro
```

Then deploy the container:

```bash
docker compose up -d
```

Follow the logs using `docker compose logs -f tcpm` to see if the miner is working correctly.
You probably need to login if this is your first time running it. Simply follow the instructions in the logs if this is the case.

### Using the binary

1. Install [Go 1.20 or later](https://go.dev/doc/install)
2. Clone the repository using [`git clone`](https://git-scm.com/docs/git-clone) (`git clone https://github.com/le0developer/go-twitch-channel-point-miner.git`) or download the repository as a ZIP file.
3. Run `go build -o tcpm .` in the TCPM directory.

Now that you have the binary, you can start using the miner.

1. Rename `tcpm.example.yaml` to `tcpm.yaml` and adjust the configuration to your needs.
2. Run `./tcpm login` to login to your Twitch account. Follow the instructions to login.
3. Then paste the login credentials to the end of the configuration file.
4. Run `./tcpm run` to start the miner.
