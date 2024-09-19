FROM golang:1.23 as build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download
COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -o main ./pkg/api

FROM alpine

WORKDIR /app

COPY --from=build /app/main .

EXPOSE 3002

ENTRYPOINT ["./main"]