FROM golang:1.14.1 as go-builder
WORKDIR /go/src/app
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-X main.GitCommit=$(git describe --tags)"

FROM debian:buster
RUN apt-get update && apt-get install -y ca-certificates && apt-get clean
COPY --from=go-builder /go/src/app/nomad-firehose /bin/
CMD [ "/bin/nomad-firehose" ]
