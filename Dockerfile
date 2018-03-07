FROM golang:1.9 as build-stage

ADD ./app /build/src/app
WORKDIR /build/src/app

#CMD /bin/nomad-firehose
ENV GOPATH=/build
ENV PATH=$PATH:$GOPATH/bin
ENV GOBUILD=$GOBUILD
ENV CGO_ENABLED=0
ENV GOOS=linux

RUN go get github.com/kardianos/govendor
RUN govendor sync
RUN go build -a -o nomad-firehose

RUN govendor list

FROM golang:1.9-alpine
COPY --from=build-stage /build/src/app/nomad-firehose /app/

ENTRYPOINT ["/app/nomad-firehose"]
