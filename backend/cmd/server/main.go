package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"transit-backend/internal/cache"
	"transit-backend/internal/config"
	"transit-backend/internal/database"
	"transit-backend/internal/handlers"
	"transit-backend/internal/hub"
	"transit-backend/internal/middleware"
	"transit-backend/internal/services"
	"transit-backend/migrations"

	"github.com/labstack/echo/v4"
)

// Compile-time interface satisfaction checks. These live in the composition
// root because it is the one place allowed to import both an interface's
// package and its concrete implementation's package without inverting the
// handlers -> services -> repositories -> database layering (CLAUDE.md
// Sec 3.2). A service or handler package importing its own concrete
// dependency here would be the layering violation DEFECT-1 was.
var (
	_ handlers.ArrivalService        = (*services.ArrivalsService)(nil)
	_ handlers.StopsService          = (*services.StopsService)(nil)
	_ handlers.RoutesService         = (*services.RoutesService)(nil)
	_ handlers.IngestService         = (*services.IngestService)(nil)
	_ handlers.NearbyService         = (*services.GeofencingService)(nil)
	_ services.StopRepository        = (*database.StopRepository)(nil)
	_ services.TripRepository        = (*database.TripRepository)(nil)
	_ services.LocationRepository    = (*database.LocationRepository)(nil)
	_ services.RouteRepository       = (*database.RouteRepository)(nil)
	_ services.DeviceCache           = (*cache.DeviceCache)(nil)
	_ services.DeviceRouteRepository = (*database.DeviceRouteRepository)(nil)
)

func main() {
	log.Println("Starting Transit Backend Server...")

	// 1. Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Config loaded: ENV=%s, DB=%s@%s:%s/%s",
		cfg.Env, cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// 2. Connect to DB
	dbPool, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()
	log.Println("Database connected")

	// 2a. Run migrations
	if err := database.RunMigrations(context.Background(), dbPool, migrations.Files); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations up to date")

	// 3. Connect to Redis
	redisClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Redis connected")

	// 4. Initialize Repositories
	locRepo := database.NewLocationRepository(dbPool)
	stopRepo := database.NewStopRepository(dbPool)
	tripRepo := database.NewTripRepository(dbPool)
	routeRepo := database.NewRouteRepository(dbPool)
	deviceRouteRepo := database.NewDeviceRouteRepository(dbPool)

	// Legacy cache wrapper for backward compatibility
	deviceCache := cache.NewDeviceCache(redisClient)

	// 5. Initialize Services
	geoService := services.NewGeofencingService(locRepo, stopRepo)
	arrivalsService := services.NewArrivalsService(stopRepo, tripRepo, locRepo, geoService)
	stopsService := services.NewStopsService(stopRepo)
	routesService := services.NewRoutesService(routeRepo)

	// 6. Initialize Hub
	wsHub := hub.New()
	go wsHub.Run()
	log.Println("WebSocket hub started")

	ingestService := services.NewIngestService(locRepo, deviceRouteRepo, deviceCache, wsHub)

	// 7. Initialize Handlers
	locHandler := handlers.NewLocationHandler(ingestService)
	routeHandler := handlers.NewRouteHandler(routesService)
	stopHandler := handlers.NewStopHandler(stopsService)
	arrivalHandler := handlers.NewArrivalHandler(arrivalsService)
	wsHandler := handlers.NewWebSocketHandler(wsHub)
	nearbyHandler := handlers.NewNearbyHandler(geoService)

	// 8. Setup Echo
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = middleware.HTTPErrorHandler

	// Middleware stack
	e.Use(middleware.Recovery)
	e.Use(middleware.CORS())
	e.Use(middleware.Logging)

	// 9. Register Routes

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// API routes
	api := e.Group("/api")

	// Location ingestion
	api.POST("/location", locHandler.IngestLocation)

	// Routes and stops
	api.GET("/routes", routeHandler.GetRoutes)
	api.POST("/routes", routeHandler.CreateRoute)
	api.GET("/stops", stopHandler.GetStops)

	// Arrivals
	api.GET("/arrivals", arrivalHandler.GetArrivals)

	// Nearby queries (H3+PostGIS)
	api.GET("/nearby/buses", nearbyHandler.GetNearbyBuses)
	api.GET("/nearby/stops", nearbyHandler.GetNearbyStops)

	// Geo utilities
	api.GET("/geo/hex", nearbyHandler.GetHexInfo)

	// WebSocket
	e.GET("/ws", wsHandler.HandleWS)

	// Serve static files
	e.Static("/", "public")

	// 10. Start Server
	go func() {
		addr := cfg.ServerHost + ":" + cfg.ServerPort
		log.Printf("Server starting on %s", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server", err)
		}
	}()

	// 11. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	log.Println("Server stopped")
}
