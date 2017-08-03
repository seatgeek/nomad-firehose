FROM alpine

# we need ca-certificates for any external https communication
RUN apk --update upgrade && \
    apk add curl ca-certificates && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*

ADD ./build/nomad-firehose-linux-amd64 /bin/nomad-firehose
ADD ./entrypoint.sh /entrypoint.sh

CMD /bin/nomad-firehose
ENTRYPOINT ["/entrypoint.sh"]
