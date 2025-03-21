FROM golang:alpine AS builder

RUN apk update && apk add --no-cache ca-certificates && update-ca-certificates

ARG BUILD_VERSION

WORKDIR $GOPATH/src/app
COPY translations translations
COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor vendor
COPY app app
COPY data data
COPY commands commands
COPY config config
COPY logger logger
COPY sender sender
COPY main.go main.go
COPY config.json /config.json
RUN go version
RUN CGO_ENABLED=0 go build -mod vendor -ldflags="-w -s -X main.version=${BUILD_VERSION}" -o /go/bin/app .

FROM scratch
WORKDIR /app/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY config.json /config.json
COPY --from=builder /go/bin/app /go/bin/app
ENTRYPOINT ["/go/bin/app"]

#
# LABEL target docker image
#

# Build arguments
ARG BUILD_ARCH
ARG BUILD_DATE
ARG BUILD_REF
ARG BUILD_VERSION

# Labels
LABEL \
    io.hass.name="telegram-delete-join-messages" \
    io.hass.description="telegram-delete-join-messages" \
    io.hass.arch="${BUILD_ARCH}" \
    io.hass.version="${BUILD_VERSION}" \
    io.hass.type="addon" \
    maintainer="ad <github@apatin.ru>" \
    org.label-schema.description="telegram-delete-join-messages" \
    org.label-schema.build-date=${BUILD_DATE} \
    org.label-schema.name="telegram-delete-join-messages" \
    org.label-schema.schema-version="1.0" \
    org.label-schema.usage="https://gitlab.com/ad/telegram-delete-join-messages/-/blob/master/README.md" \
    org.label-schema.vcs-ref=${BUILD_REF} \
    org.label-schema.vcs-url="https://github.com/ad/telegram-delete-join-messages/" \
    org.label-schema.vendor="HomeAssistant add-ons by ad"
