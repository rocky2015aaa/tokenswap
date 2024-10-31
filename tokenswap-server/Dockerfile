FROM golang:1.22 AS base

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN CGO_ENABLED=0 go build -o stsvr cmd/stsvr/main.go

FROM alpine:latest

RUN mkdir /appication 
WORKDIR /appication

COPY --from=base /app/stsvr .
COPY --from=base /app/.env_dev .
COPY --from=base /app/docker-compose-dev.yml .

EXPOSE 9081

CMD ["./stsvr"]

