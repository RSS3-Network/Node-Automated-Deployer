# Node Automated Deployer

This Deployer automatically deploys an RSS3 DSL Node based on a `config.yaml` file.


## Usage

### Download

download the latest release from [release page](https://github.com/RSS3-Network/Node-Automated-Deployer/releases)

```bash
tar -zxvf node-automated-deployer-v0.1.0-linux-amd64.tar.gz
``` 

### Configuration

create a `config.yaml` file in the subdirectory `config` of the executable file.

### Generate

```bash
./node-automated-deployer > docker-compose.yaml
```

