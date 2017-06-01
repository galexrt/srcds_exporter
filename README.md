# srcds_exporter
SRCDS Gameserver Prometheus exporter.

[![CircleCI branch](https://img.shields.io/circleci/project/github/RedSparr0w/node-csgo-parser/master.svg)]() [![Docker Repository on Quay](https://quay.io/repository/galexrt/srcds_exporter/status "Docker Repository on Quay")](https://quay.io/repository/galexrt/srcds_exporter) [![Go Report Card](https://goreportcard.com/badge/github.com/galexrt/srcds_exporter)](https://goreportcard.com/report/github.com/galexrt/srcds_exporter) [![license](https://img.shields.io/github/license/mashape/apistatus.svg)]()

## Collectors
Whick collectors are enabled is controlled by the `--collectors.enabled` flag.

### Enabled by default

Name     | Description
---------|-------------
playercount | Current player count
map | Current map played

### Disabled by default

Name     | Description
---------|-------------
players | Report all players by their Steam ID as a metric

## Usage
To get a list of a list of all flags, use the `--help` flag.

## Docker Image
The Docker image is available from [Quay.io](https://quay.io):
* `quay.io/galexrt/srcds_exporter:1.0.0`
