package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"transit-backend/internal/cache"
	"transit-backend/internal/config"
	"transit-backend/internal/database"
	"transit-backend/internal/handlers"
	"transit-backend/internal/hub"
	"transit-backend/internal/middleware"

	"github.com/labstack/echo/v4"
)

func main() {
	// 1. Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Connect to DB
	dbPool, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// 3. Connect to Redis
	redisClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	// 4. Initialize Repositories
	locRepo := database.NewLocationRepository(dbPool)
	stopRepo := database.NewStopRepository(dbPool)
	tripRepo := database.NewTripRepository(dbPool)
	routeRepo := database.NewRouteRepository(dbPool)

	deviceCache := cache.NewDeviceCache(redisClient)

	// 5. Initialize Hub
	wsHub := hub.New()
	go wsHub.Run()

	// 6. Initialize Handlers
	locHandler := handlers.NewLocationHandler(locRepo, deviceCache, wsHub)
	routeHandler := handlers.NewRouteHandler(routeRepo)
	stopHandler := handlers.NewStopHandler(stopRepo)
	arrivalHandler := handlers.NewArrivalHandler(stopRepo, tripRepo)
	wsHandler := handlers.NewWebSocketHandler(wsHub)

	// 7. Setup Echo
	e := echo.New()
	e.HTTPErrorHandler = middleware.HTTPErrorHandler
	e.Use(middleware.CORS())
	e.Use(middleware.Logging)

	// 8. Register Routes
	api := e.Group("/api")
	api.POST("/location", locHandler.IngestLocation)
	api.GET("/routes", routeHandler.GetRoutes)
	api.GET("/stops", stopHandler.GetStops)
	api.GET("/arrivals", arrivalHandler.GetArrivals)

	e.GET("/ws", wsHandler.HandleWS)

	// Serve static files
	e.Static("/", "public")

	// 9. Start Server
	go func() {
		addr := cfg.ServerHost + ":" + cfg.ServerPort
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
