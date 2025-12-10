ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /usr/src/app

COPY . /usr/src/app/

RUN go build -o hackney-bindicator app.go

FROM debian:bookworm

WORKDIR /

COPY --from=builder /usr/src/app/hackney-bindicator /app/
COPY --from=builder /usr/src/app/README.md /app/
COPY --from=builder /usr/src/app/static /app/static/

CMD [ "/app/hackney-bindicator" ]