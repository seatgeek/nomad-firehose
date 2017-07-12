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

The allocation output is different from the [default API response](https://www.nomadproject.io/docs/http/allocs.html), as the tool will emit an event per new [TaskStates](https://www.nomadproject.io/docs/http/allocs.html), rather than all the previous events.

```json
{
    "Name": "job.task[0]",
    "AllocationID": "1ef2eba2-00e4-3828-96d4-8e58b1447aaf",
    "DesiredStatus": "run",
    "DesiredDescription": "",
    "ClientStatus": "running",
    "ClientDescription": "",
    "JobID": "logrotate",
    "GroupName": "cron",
    "TaskName": "logrotate",
    "EvalID": "bf926150-ed30-6c13-c597-34d7a3165fdc",
    "TaskState": "running",
    "TaskFailed": false,
    "TaskStartedAt": "2017-06-30T19:58:28.325895579Z",
    "TaskFinishedAt": "0001-01-01T00:00:00Z",
    "TaskEvent": {
        "Type": "Task Setup",
        "Time": 1498852707712617200,
        "FailsTask": false,
        "RestartReason": "",
        "SetupError": "",
        "DriverError": "",
        "DriverMessage": "",
        "ExitCode": 0,
        "Signal": 0,
        "Message": "Building Task Directory",
        "KillReason": "",
        "KillTimeout": 0,
        "KillError": "",
        "StartDelay": 0,
        "DownloadError": "",
        "ValidationError": "",
        "DiskLimit": 0,
        "DiskSize": 0,
        "FailedSibling": "",
        "VaultError": "",
        "TaskSignalReason": "",
        "TaskSignal": ""
    }
}
```

### `nodes`

`nomad-firehose nodes` will monitor all node changes in the Nomad cluster and emit an firehose event per change to the configured sink.

The output will be equal to the [Nomad Node API structure](https://www.nomadproject.io/docs/http/node.html)

### `evaluations`

`nomad-firehose evaluations` will monitor all evaluation changes in the Nomad cluster and emit an firehose event per change to the configured sink.

The output will be equal to the [Nomad Evaluation API structure](https://www.nomadproject.io/docs/http/eval.html)

### `jobs`

`nomad-firehose jobs` will monitor all job changes in the Nomad cluster and emit an firehose event per change to the configured sink.

The output will be equal to the *full* [Nomad Job API structure](https://www.nomadproject.io/docs/http/job.html)
