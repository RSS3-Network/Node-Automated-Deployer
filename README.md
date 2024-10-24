# Node Automated Deployer

This Deployer automatically deploys an RSS3 DSL Node based on a `config.yaml` file.

For more information, please refer to the [RSS3 Node Deployment Guide](https://docs.rss3.io/docs/node).

## Automated Deployment

```bash
curl -s https://raw.githubusercontent.com/RSS3-Network/Node-Automated-Deployer/main/automated_deploy.sh | sudo bash
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

### Upgrade

If you are upgrading from an older version, please replace the `config.yaml` file with the new one (generated from [Explorer](https://explorer.rss3.io/)) or modify the configurations according to the [Deployment Guide](https://docs.rss3.io/guide/operator/deployment/guide#configuration-options).

#### Upgrading from v1.0.x or older to v1.1.x

```yaml
component:
    rss:
        id: rsshub-core
        network: rsshub
        worker: core
```

`rss` component has breaking changes:

- `id`, from `rss-rsshub` to `rsshub-core`
- `network`, from `rss` to `rsshub`
- `worker`, from `rsshub` to `core`

Other components remain the same.

### Deploy

```bash
./node-automated-deployer > docker-compose.yaml
```

```bash
docker-compose up -d
```
