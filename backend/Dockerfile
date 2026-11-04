FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl wget

COPY bin/server /app/server

EXPOSE 8080

CMD ["/app/server"]

