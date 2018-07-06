# nomad-firehose

`nomad-firehose` is a tool meant to enable teams to quickly build logic around nomad task events without hooking into Nomad API.

## Running

The project has build artifacts for Linux, Darwin and Windows in the [GitHub releases tab](https://github.com/seatgeek/nomad-firehose/releases).

A Docker container is also provided at [seatgeek/nomad-firehose](https://hub.docker.com/r/seatgeek/nomad-firehose/tags/)

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

Any `CONSUL_*` env that the native `consul` CLI tool supports are supported by this tool. Additionally, the variables `CONSUL_LOCK_PREFIX` (default `nomad-firehose/`) and `CONSUL_SESSION_NAME` (default `nomad-firehose-allocations`) control where leader locks are stored.

The most basic requirement is `export NOMAD_ADDR=http://<ip>:4646` and `export CONSUL_HTTP_ADDR=<ip>:8500`.

### Consul

`nomad-firehose` will use Consul to maintain leader-ship and store last event time processed (saved on quit or every 10s).

This mean you can run more than 1 process of each firehose, and only one will actually do any work.

Saving the last event time mean that restarting the process won't firehose all old changes to your sink, reducing duplicated events.

The Consul lock is maintained in KV at `$CONSUL_LOCK_PREFIX/${type}.lock` and the last event time is stored in KV at `$CONSUL_LOCK_PREFIX/${type}.value`.

#### Consul ACL Token Permissions

If the Consul cluster being used is running ACLs, the following ACL policy will allow the required access, given the default values for `CONSUL_LOCK_PREFIX` and `CONSUL_SESSION_NAME`:

```hcl
key "nomad-firehose" {
  policy = "write"
}
session "" {
  policy = "write"
}
```

## Usage

The `nomad-firehose` binary has several helper subcommands.

The sink type is configured using `$SINK_TYPE` environment variable. Valid values are:
- `amqp`
- `kinesis`
- `nsq`
- `redis`
- `stdout`

The `amqp` sink is configured using `$SINK_AMQP_CONNECTION` (`amqp://guest:guest@127.0.0.1:5672/`), `$SINK_AMQP_EXCHANGE` and `$SINK_AMQP_ROUTING_KEY`, `$SINK_AMQP_WORKERS` (default: `1`) environment variables.

The `kinesis` sink is configured using `$SINK_KINESIS_STREAM_NAME` and `$SINK_KINESIS_PARTITION_KEY` environment variables.

The `nsq` sink is configured using `$SINK_NSQ_ADDR` and `$SINK_NSQ_TOPIC_NAME` environment variables.

The `redis` sink is configured using `$SINK_REDIS_URL` (`redis://[user]:[password]@127.0.0.1[:5672]/0`) and `$SINK_REDIS_KEY` environment variables.

The `kafka` sink is configured using `$SINK_KAFKA_BROKERS` (`kafka1:9092,kafka2:9092,kafka3:9092`), and `$SINK_KAFKA_TOPIC` environment variables.

The `stdout` sink does not have any configuration, it will simply output the JSON to stdout for debugging.

### `allocations`

`nomad-firehose allocations` will monitor all allocation changes in the Nomad cluster and emit each task state as a new firehose event to the configured sink.

The allocation output is different from the [default API response](https://www.nomadproject.io/api/allocations.html), as the tool will emit an event per new [TaskStates](https://www.nomadproject.io/docs/http/allocs.html), rather than all the previous events.

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

`nomad-firehose nodes` will monitor all node changes in the Nomad cluster and emit a firehose event per change to the configured sink.

The output will be equal to the [Nomad Node API structure](https://www.nomadproject.io/api/nodes.html)

### `evaluations`

`nomad-firehose evaluations` will monitor all evaluation changes in the Nomad cluster and emit a firehose event per change to the configured sink.

The output will be equal to the [Nomad Evaluation API structure](https://www.nomadproject.io/api/evaluations.html)

### `jobs`

`nomad-firehose jobs` will monitor all job changes in the Nomad cluster and emit a firehose event per change to the configured sink.

The output will be equal to the *full* [Nomad Job API structure](https://www.nomadproject.io/api/jobs.html)

### `deployments`

`nomad-firehose deployments` will monitor all deployment changes in the Nomad cluster and emit a firehose event per change to the configured sink.

The output will be equal to the *full* [Nomad Deployment API structure](https://www.nomadproject.io/api/deployments.html)
