# srcds_exporter
SRCDS Gameserver Prometheus exporter.

[![CircleCI branch](https://img.shields.io/circleci/project/github/RedSparr0w/node-csgo-parser/master.svg)]() [![Docker Repository on Quay](https://quay.io/repository/galexrt/srcds_exporter/status "Docker Repository on Quay")](https://quay.io/repository/galexrt/srcds_exporter) [![Go Report Card](https://goreportcard.com/badge/github.com/galexrt/srcds_exporter)](https://goreportcard.com/report/github.com/galexrt/srcds_exporter) [![license](https://img.shields.io/github/license/mashape/apistatus.svg)]()

## Compatibility

### Tested Games
* [Garry's Mod](https://store.steampowered.com/app/4000/Garrys_Mod/)

It may work with newer Source Engine games like [Counter-Strike: Global Offensive](http://store.steampowered.com/app/730/CounterStrike_Global_Offensive/) too.
If you have any issues with a game, please let me know by creating an issue containing the rcon output of `status` command and I'll be glad to fix it.

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
* `quay.io/galexrt/srcds_exporter:v1.0.1`
