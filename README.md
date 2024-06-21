# Coding Challenge: AI Immigration Chatbot with Golang and Temporal


## Start the platform

```
docker-compose up -d
```

## Start your http server
```
go run main.go
```

## Test it first!
```
curl -X POST http://localhost:3002/chat -d '{"user":"TOTO","content":"visa"}'
```