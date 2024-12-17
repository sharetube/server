FROM golang:1.23.4-alpine3.20 AS build

RUN apk update && \
    apk upgrade && \
    apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM scratch

WORKDIR /app

COPY --from=build app/server .

CMD ["./server"]