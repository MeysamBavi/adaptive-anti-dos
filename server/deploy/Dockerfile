FROM golang:1.22.5 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY server/ ./server

RUN go build -o server-bin ./server/main.go

FROM debian:12.5

WORKDIR /app

COPY --from=build /app/server-bin .
COPY --from=build /app/server/assets /files

EXPOSE 8080

CMD ["./server-bin"]
