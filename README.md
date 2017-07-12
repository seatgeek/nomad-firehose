# nomad-firehose

`nomad-firehose` is a tool meant to enable teams to quickly build logic around nomad task events without hooking into Nomad API.

## Running

The project got build artifacts for linux, darwin and windows in the [GitHub releases tab](https://github.com/seatgeek/nomad-firehose/releases).

A docker container is also provided at [seatgeek/nomad-firehose](https://hub.docker.com/r/seatgeek/nomad-firehose/tags/)

## Requirements

- Go 1.8

## Building

To build a binary, run the following

```shell
# get this repo
go get github.com/seatgeek/nomad-firehose

# go to the repo directory
cd $GOPATH/src/github.com/seatgeek/nomad-firehose

# build the `nomad-firehose` binary
make build
```

This will create a `nomad-firehose` binary in your `$GOPATH/bin` directory.

## Configuration

Any `NOMAD_*` env that the native `nomad` CLI tool supports are supported by this tool.

Any `CONSUL_*` env that the native `consul` CLI tool supports are supported by this tool.

The most basic requirement is `export NOMAD_ADDR=http://<ip>:4646` and `export CONSUL_HTTP_ADDR=<ip>:8500`.

## Usage

The `nomad-firehose` binary has several helper subcommands.

The sink type is specified via the `$SINK_TYPE` environment variable. Valid values are: `stdout`, `kinesis` and `amqp`.

The `amqp` sink is configured using `$SINK_AMQP_CONNECTION`, `$SINK_AMQP_EXCHANGE` and `$SINK_AMQP_ROUTING_KEY` environment variables.

The `kinesis` sink is configured using `$SINK_KINESIS_STREAM_NAME` and `$SINK_KINESIS_PARTITION_KEY` environment variables.

The `stdout` sink do not have any configuration.

The script will use Consul to maintain leader and the last event time processed (saved on quit or every 10s).

### `allocations`

`nomad-firehose allocations` will monitor all alocation changes in the Nomad cluster and emit each task state as a new firehose event to the configured sink.

### `nodes`

`nomad-firehose nodes` will monitor all node changes in the Nomad cluster and emit an firehose event per change to the configured sink.

### `evaluations`

`nomad-firehose evaluations` will monitor all evaluation changes in the Nomad cluster and emit an firehose event per change to the configured sink.

### `jobs`

`nomad-firehose jobs` will monitor all job changes in the Nomad cluster and emit an firehose event per change to the configured sink.
