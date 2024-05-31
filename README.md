# Node Automated Deployer

This Deployer automatically deploys an RSS3 DSL Node based on a `config.yaml` file.

For more information, please refer to the [RSS3 Node Deployment Guide](https://docs.rss3.io/docs/node).

## Automated Deployment

```bash
curl -s https://raw.githubusercontent.com/RSS3-Network/Node-Automated-Deployer/main/node-automated-deployer/automated_deploy.sh | bash
```

```bash
chmod -x automated_deploy.sh
```

And you are done!

## Manual Deployment

### Download

Download the latest release from [release page](https://github.com/RSS3-Network/Node-Automated-Deployer/releases)

```bash
tar -zxvf downloaded_file.tar.gz
``` 

### Configuration

Your `config.yaml` must be placed in the `config` subdirectory, at the same level as the `node-automated-deployer` script.

### Deploy

```bash
./node-automated-deployer > docker-compose.yaml
```

```bash
docker-compose up -d
```

