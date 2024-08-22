FROM golang:1.22.5 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY controller/ ./controller

RUN go build -o controller-bin ./controller/main.go

FROM debian:12.5

WORKDIR /app

COPY --from=build /app/controller-bin .

EXPOSE 8080

CMD ["./controller-bin"]
