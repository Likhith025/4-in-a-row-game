package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/connect-four/internal/api"
	"github.com/connect-four/internal/game"
	"github.com/connect-four/internal/kafka"
	"github.com/connect-four/internal/matchmaker"
	"github.com/connect-four/internal/storage"
	"github.com/connect-four/internal/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return // .env file is optional
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" { // Don't override existing env vars
				os.Setenv(key, value)
			}
		}
	}
}

func main() {
	// Load .env file if present
	loadEnvFile(".env")
	
	ctx := context.Background()

	// Initialize PostgreSQL store
	store, err := storage.NewPostgresStore(ctx)
	if err != nil {
		log.Printf("Warning: Database not available: %v", err)
		log.Println("Running in memory-only mode (games won't be persisted)")
		store = nil
	} else {
		defer store.Close()
	}

	// Initialize Kafka producer
	producer, err := kafka.NewProducer()
	if err != nil {
		log.Printf("Warning: Kafka producer not available: %v", err)
	}
	defer producer.Close()

	// Initialize Kafka consumer (optional)
	var consumer *kafka.Consumer
	if producer.IsEnabled() {
		consumer, err = kafka.NewConsumer()
		if err != nil {
			log.Printf("Warning: Kafka consumer not available: %v", err)
		} else {
			consumer.Start()
			defer consumer.Stop()
		}
	}

	// Initialize matchmaker
	mm := matchmaker.NewMatchmaker()

	// Initialize WebSocket hub
	hub := websocket.NewHub(mm)

	// Set up game start callback for Kafka events
	mm.SetOnGameStart(func(g *game.Game) {
		producer.EmitGameStart(g)
	})

	// Set up game end callback for persistence and Kafka
	hub.SetOnGameEnd(func(g *game.Game) {
		// Emit Kafka event
		producer.EmitGameEnd(g)

		// Persist to database
		if store != nil {
			if err := store.SaveGame(context.Background(), g); err != nil {
				log.Printf("Error saving game: %v", err)
			}
		}
	})

	// Start WebSocket hub
	go hub.Run()

	// Create message handler
	handler := websocket.NewHandler(hub, mm)

	// Set up HTTP router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API routes
	r.Route("/api", func(r chi.Router) {
		apiHandlers := api.NewHandlers(store, mm, producer, consumer)
		apiHandlers.RegisterRoutes(r)
	})

	// WebSocket endpoint
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(hub, handler, w, r)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Get port from environment or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		log.Printf("WebSocket endpoint: ws://localhost:%s/ws", port)
		log.Printf("API endpoint: http://localhost:%s/api", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
