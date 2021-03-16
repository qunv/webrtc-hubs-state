FROM golang:alpine as builder

LABEL maintainer="quannv@dev"

RUN apk update

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/

EXPOSE 8888

CMD ["./main"]
