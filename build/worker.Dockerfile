FROM golang:1.23 as build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download
COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -o main ./pkg/workers

FROM alpine

WORKDIR /app

COPY --from=build /app/main .

EXPOSE 7239 9090

ENTRYPOINT ["./main"]
