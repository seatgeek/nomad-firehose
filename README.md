# nomad-firehose

`nomad-firehose` is a tool meant to enable teams to quickly build logic around nomad task events without hooking into Nomad API.

## Running

The project has build artifacts for Linux, Darwin and Windows in the [GitHub releases tab](https://github.com/seatgeek/nomad-firehose/releases).

A Docker container is also provided at [seatgeek/nomad-firehose](https://hub.docker.com/r/seatgeek/nomad-firehose/tags/)

## Requirements

- Go 1.11

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

### Consul

`nomad-firehose` will use Consul to maintain leader-ship and store last event time processed (saved on quit or every 10s).

This mean you can run more than 1 process of each firehose, and only one will actually do any work.

Saving the last event time mean that restarting the process won't firehose all old changes to your sink, reducing duplicated events.

By default, the Consul lock is maintained in KV at `nomad-firehose/${type}.lock` and the last event time is stored in KV at `nomad-firehose/${type}.value`. You can change the prefix from `nomad-firehose` by setting `NOMAD_FIREHOSE_CONSUL_PREFIX` to your desired prefix.

#### Consul ACL Token Permissions

If the Consul cluster being used is running ACLs, the following ACL policy will allow the required access:

```hcl
key "nomad-firehose" {
  policy = "write"
}
session "" {
  policy = "write"
}
```

If you've set a custom prefix, specify that in the `key` ACL entry instead.

### Kafka

To connect to Kafka with TLS, set the SINK_KAFKA_CA_CERT_PATH to the path to your CA cert file.
To use SASL/PLAIN authentication, set `$SINK_KAFKA_USER` and `$SINK_KAFKA_PASSWORD` environment variables.


## Usage

The `nomad-firehose` binary has several helper subcommands.

The sink type is configured using `$SINK_TYPE` environment variable. Valid values are:
- `amqp`
- `kinesis`
- `nsq`
- `redis`
- `kafka`
- `mongo`
- `stdout`
- `syslog`

The `amqp` and `rabbitmq` sinks are configured using `$SINK_AMQP_CONNECTION` (`amqp://guest:guest@127.0.0.1:5672/`), `$SINK_AMQP_EXCHANGE`, `$SINK_AMQP_ROUTING_KEY`, and `$SINK_AMQP_WORKERS` (default: `1`) environment variables.

The `http` sink is configured using `$SINK_HTTP_ADDRESS` (`localhost:8080/allocations`)` environment variable.

The `kafka` sink is configured using `$SINK_KAFKA_BROKERS` (`kafka1:9092,kafka2:9092,kafka3:9092`), and `$SINK_KAFKA_TOPIC` environment variables.

The `kinesis` sink is configured using `$SINK_KINESIS_STREAM_NAME` and `$SINK_KINESIS_PARTITION_KEY` environment variables.

The `mongo` sink is configured using `$SINK_MONGODB_CONNECTION` (`mongodb://localhost:27017/`), `$SINK_MONGODB_DATABASE` and `$SINK_MONGODB_COLLECTION` environment variables.

The `nsq` sink is configured using `$SINK_NSQ_ADDR` and `$SINK_NSQ_TOPIC_NAME` environment variables.

The `redis` sink is configured using `$SINK_REDIS_URL` (`redis://[user]:[password]@127.0.0.1[:5672]/0`) and `$SINK_REDIS_KEY` environment variables.

The `stdout` sink does not have any configuration, it will simply output the JSON to stdout for debugging.

The `syslog` sink is configured using `$SINK_SYSLOG_PROTO` (e.g. `tcp`, `udp` - leave empty if logging to a local syslog socket), `$SINK_SYSLOG_ADDR` (e.g. `127.0.0.1:514` - leave empty if logging to a local syslog socket), and `$SINK_SYSLOG_TAG` (default: `nomad-firehose`).

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

The output will be equal to the *full* [Nomad Job API structure](https://www.nomadproject.io/api/jobs.html#read-job)

### `jobliststubs`

`nomad-firehose jobliststubs` will monitor all job changes in the Nomad cluster and emit a firehose event per change to the configured sink.

The output will be equal to the job list [Nomad Job API structure](https://www.nomadproject.io/api/jobs.html#list-jobs)

### `deployments`

`nomad-firehose deployments` will monitor all deployment changes in the Nomad cluster and emit a firehose event per change to the configured sink.

The output will be equal to the *full* [Nomad Deployment API structure](https://www.nomadproject.io/api/deployments.html)
