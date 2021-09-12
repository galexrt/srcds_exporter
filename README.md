# srcds_exporter

![build_release](https://github.com/galexrt/srcds_exporter/workflows/build_release/badge.svg)

SRCDS Gameserver Prometheus exporter.

Container Image available from:

* [Quay.io](https://quay.io/repository/galexrt/srcds_exporter)
* [GHCR.io](https://github.com/users/galexrt/packages/container/package/srcds_exporter)

Container Image Tags:

* `GIT_TAG` - Each Git tag is built and published.

## Compatibility

### Tested Games

* [Garry's Mod](https://store.steampowered.com/app/4000/Garrys_Mod/)
* [Counter-Strike: Source](https://store.steampowered.com/app/240/CounterStrike_Source/)
* [Counter-Strike: Global Offensive](https://store.steampowered.com/app/730/CounterStrike_Global_Offensive/)

If you have any issues with a game, please create an issue containing the rcon output of `status` command and we'll see what we can do to fix compatibility.

## Collectors

A collector is collecting certain metrics. Which collectors are enabled is controlled by the `--collectors.enabled` flag.

### Enabled by default

| Name          | Description          |
| ------------- | -------------------- |
| `playercount` | Current player count |
| `map`         | Current map played   |

### Disabled by default

| Name      | Description                                                  |
| --------- | ------------------------------------------------------------ |
| `players` | Report all players by with their Steam ID label as a metric. |

## Usage

Create the `srcds_exporter` config file (see [srcds.example.yml](srcds.example.yml) for an example). The config file can be named whatever you want, the path to the config must be passed to the `srcds_exporter` through the `-config.file=FILE_PATH` flag (default: `./srcds.yaml` (current directoy file `srcds.yaml`)).

Then just run the `srcds_exporter` binary, through Docker (don't forget to add a mount so the config is available in the container), directly or by having it in your `PATH`.

### Flags

To get a list of all available flags, use the `--help` flag (e.g., `srcds_exporter --help`).

Example output:

```shell
$ srcds_exporter --help
Usage of srcds_exporter:
      --collectors.enabled string   Comma separated list of active collectors (default "map,playercount")
      --collectors.print            If true, print available collectors and exit.
      --config.file string          Config file to use. (default "./srcds.yaml")
      --log-level string            Set log level (default "INFO")
      --version                     Show version information
      --web.listen-address string   The address to listen on for HTTP requests (default ":9137")
      --web.telemetry-path string   Path the metrics will be exposed under (default "/metrics")
pflag: help requested
exit status 2
```
