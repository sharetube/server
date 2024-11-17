FROM golang:1.23.2-alpine3.20 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM scratch

WORKDIR /app

COPY --from=build app/server .

ENTRYPOINT ["./server"]