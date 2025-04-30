# go-twitch-channel-point-miner

Channel point miner for Twitch written in Go.

## Usage

### Using Docker

Download the `tcpm.example.yaml` and rename it to `tcpm.yaml`:

```bash
wget https://raw.githubusercontent.com/le0developer/go-twitch-channel-point-miner/master/tcpm.example.yaml -O tcpm.yaml
```

Adjust the configuration to your needs.

Create an empty persistence file: `echo "{}" >> persistent.json`.

Create a `docker-compose.yml` file:

```yaml
services:
  tcpm:
    image: ghcr.io/le0developer/go-twitch-channel-point-miner:master
    restart: unless-stopped
    command: ["run", "--login"]
    volumes:
      - ./tcpm.yaml:/tcpm.yaml
      - ./persistent.json:/persistent.json
      # The container does not have its own certificate store, so you'll need to mount the host's certificate store
      - /etc/ssl/certs:/etc/ssl/certs:ro
```

Then deploy the container:

```bash
docker compose up -d
```

Follow the logs using `docker compose logs -f tcpm` to see if the miner is working correctly.
You probably need to login if this is your first time running it. Simply follow the instructions in the logs if this is the case.

##### Updating

To update the miner, simply run `docker compose pull` and then `docker compose up -d` again.

### Using `go install`

1. Install [Go 1.20 or later](https://go.dev/doc/install)
2. Run `go install github.com/le0developer/go-twitch-channel-point-miner@latest`
3. The executable will be globally available as `go-twitch-channel-point-miner`

Now follow the instructions in the [Configuration](#configuration) section to configure the miner.

##### Updating

To update the miner, simply run `go install github.com/le0developer/go-twitch-channel-point-miner@latest` again.

### Using the binary

1. Install [Go 1.20 or later](https://go.dev/doc/install)
2. Clone the repository using [`git clone`](https://git-scm.com/docs/git-clone) (`git clone https://github.com/le0developer/go-twitch-channel-point-miner.git`) or download the repository as a ZIP file.
3. Run `go build .` in the TCPM directory.
4. The executable will be available as `go-twitch-channel-point-miner` in the current directory.

Now follow the instructions in the [Configuration](#configuration) section to configure the miner.

##### Updating

If you cloned the repository, simply run `git pull` to update the repository and then run `go build .` again to build the latest version.  
If you downloaded the ZIP file, you will need to download it again and build it again.

## Configuration

Download the [`tcpm.example.yaml`](./tcpm.example.yaml) file and place as `tcpm.yaml` it in your current directory.

Go through the configuration file and adjust the settings to your needs.

## Logging in

Run `./go-twitch-channel-point-miner login -u {TWITCH_USERNAME}` to login to your Twitch account.
This will prompt you to go to `https://www.twitch.tv/activate` and enter a code.

After authorization, the miner will spit out your login credentials in the terminal.
Append these credentials to the end of the `tcpm.yaml` file.

## Notificatons

Work in progress.
