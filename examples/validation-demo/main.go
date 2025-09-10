package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/flowgraph/flowgraph/pkg/validation"
)

type Address struct {
	Street string `json:"street" validate:"required"`
	City   string `json:"city" validate:"required"`
	Zip    string `json:"zip" validate:"required,len=5"`
}

type RegistrationRequest struct {
	Name    string  `json:"name" validate:"required,min=2,max=50"`
	Email   string  `json:"email" validate:"required,email"`
	Age     int     `json:"age" validate:"min=18,max=120"`
	Address Address `json:"address" validate:"required"`
}

func main() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request from %s", r.Method, r.RemoteAddr)
		var req RegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		log.Printf("Request body: %+v", req)
		if err := validation.ValidateStruct(req); err != nil {
			log.Printf("Validation error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"errors": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]string{"message": "Registration successful"}
		log.Printf("Response: %v", resp)
		json.NewEncoder(w).Encode(resp)
	})

	server := &http.Server{
		Addr:    ":8081",
		Handler: handler,
	}

	log.Printf("User Registration API server running at %s", server.Addr)
	log.Println("Send POST requests with JSON body to test validation.")
	log.Println("Example curl:")
	log.Println(`curl -X POST http://localhost:8081 -H 'Content-Type: application/json' -d '{"name":"Alice Smith","email":"alice@example.com","age":30,"address":{"street":"123 Main St","city":"Metropolis","zip":"12345"}}'`)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server exited: %v", err)
	}
}
