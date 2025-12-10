ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /usr/src/app

COPY . /usr/src/app/

RUN go build -o hackney-bindicator app.go

FROM debian:bookworm

WORKDIR /

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/src/app/hackney-bindicator /app/
EXPOSE 8080

CMD [ "/app/hackney-bindicator" ]