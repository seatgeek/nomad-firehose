FROM golang:1.11-alpine

# Adding ca-certificates for external communication and git for dep installation
RUN apk add --update ca-certificates git && rm -rf /var/cache/apk/*

RUN go get -u github.com/golang/dep/cmd/dep
WORKDIR /go/src/github.com/seatgeek/nomad-firehose/
COPY . /go/src/github.com/seatgeek/nomad-firehose/
RUN dep ensure -vendor-only
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/nomad-firehose -ldflags "-X main.GitCommit=$(git describe --tags)"

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=0 /go/src/github.com/seatgeek/nomad-firehose/build/nomad-firehose /usr/local/bin/
CMD [ "nomad-firehose" ]
