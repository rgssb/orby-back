FROM golang:1.26-alpine AS build
ENV GOTOOLCHAIN=auto CGO_ENABLED=0
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /orby-api .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 orby
WORKDIR /app
COPY --from=build /orby-api .
RUN mkdir -p /data/midia && chown orby /data/midia
USER orby
ENV MEDIA_DIR=/data/midia
VOLUME /data/midia
EXPOSE 9000
CMD ["./orby-api"]
