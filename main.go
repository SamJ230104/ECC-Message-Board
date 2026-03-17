package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	database "MessageBoard/Database"
	handlers "MessageBoard/Handlers"
	middleware "MessageBoard/Middleware"
)

func main() {
	db, err := database.InitDB("")
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}

	defer db.Close()

	log.Println("✓ Database initialized and migrated")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Message board API running")
	})
	mux.HandleFunc("POST /register", handlers.Register(db))
	mux.HandleFunc("POST /login", handlers.Login(db))

	mux.HandleFunc("POST /logout", middleware.Auth(db, handlers.Logout(db)))
	mux.HandleFunc("GET /users/{username}", middleware.Auth(db, handlers.GetUser(db)))
	mux.HandleFunc("POST /messages/public", middleware.Auth(db, handlers.PostPublicMessage(db)))
	mux.HandleFunc("GET /messages/public", middleware.Auth(db, handlers.GetPublicMessages(db)))
	mux.HandleFunc("POST /messages/private", middleware.Auth(db, handlers.SendPrivateMessage(db)))
	mux.HandleFunc("GET /messages/private", middleware.Auth(db, handlers.SendPrivateMessage(db)))
	mux.HandleFunc("GET /message/private/{username}", middleware.Auth(db, handlers.GetConversation(db)))
	mux.HandleFunc("PATCH /messages/private/{id}/read", middleware.Auth(db, handlers.MarkMessageRead(db)))

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe failed: %v", err)
		}
	}()

	<-done
	log.Println("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
