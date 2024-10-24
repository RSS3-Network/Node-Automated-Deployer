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

### Upgrading

One can check what has changed in the [Changelog](https://github.com/RSS3-Network/Node/releases).

ℹ️ When upgrading, please ensure to update your `config.yaml` file or modify the configurations according to the [Deployment Guide](https://docs.rss3.io/guide/operator/deployment/guide#configuration-options).

⚠️ Please read the release notes carefully before upgrading, especially for major version changes.

New major versions may contain incompatible breaking changes.

#### Upgrading from v1.0.x to v1.1.x

When upgrading from v1.0.x or older to v1.1.x, the `rss` component has breaking changes:

```diff
component:
    rss:
---     id: rss-rsshub
+++     id: rsshub-core
---     network: rss
+++     network: rsshub
---     worker: rsshub
+++     worker: core
```

You should update your `config.yaml` file accordingly. Other components remain the same.

### Deploy

```bash
./node-automated-deployer > docker-compose.yaml
```

```bash
docker-compose up -d
```
