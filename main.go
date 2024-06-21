package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Message struct {
	User    string `json:"user"`
	Content string `json:"content"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	var msg Message
	_ = json.NewDecoder(r.Body).Decode(&msg)
	json.NewEncoder(w).Encode(msg)
}

func main() {
	http.HandleFunc("/chat", handler)
	log.Println("Server started at http://localhost:3002")
	log.Fatal(http.ListenAndServe(":3002", nil))
}
