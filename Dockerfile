FROM alpine

ENV CONSUL_HTTP_ADDR "consul.service.owf-live:8500"
ENV NOMAD_ADDR 'http://nomad.service.owf-live:4646'
ENV SINK_TYPE d3slack
ENV D3_SLACK_WEBHOOK "https://hooks.slack.com/services/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"
ENV D3_BOT_NAME "TestBot"

# we need ca-certificates for any external https communication
RUN apk --update upgrade && \
    apk add curl ca-certificates && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*

COPY build/nomad-firehose-linux-amd64 /bin/nomad-firehose
COPY entrypoint.sh /entrypoint.sh

CMD /bin/nomad-firehose
ENTRYPOINT ["/entrypoint.sh"]
