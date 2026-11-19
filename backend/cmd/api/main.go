package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	auditrepo "buffalo-pos/internal/audit/adapter/mysqlrepo"
	authhttp "buffalo-pos/internal/auth/adapter/httpapi"
	authrepo "buffalo-pos/internal/auth/adapter/mysqlrepo"
	authservice "buffalo-pos/internal/auth/service"
	cataloghttp "buffalo-pos/internal/catalog/adapter/httpapi"
	catalogrepo "buffalo-pos/internal/catalog/adapter/mysqlrepo"
	catalogservice "buffalo-pos/internal/catalog/service"
	ordershttp "buffalo-pos/internal/orders/adapter/httpapi"
	ordersrepo "buffalo-pos/internal/orders/adapter/mysqlrepo"
	ordersservice "buffalo-pos/internal/orders/service"
	"buffalo-pos/internal/platform/config"
	"buffalo-pos/internal/platform/httpx"
	"buffalo-pos/internal/platform/ratelimit"
	tableshttp "buffalo-pos/internal/tables/adapter/httpapi"
	tablesrepo "buffalo-pos/internal/tables/adapter/mysqlrepo"
	tablesservice "buffalo-pos/internal/tables/service"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	log.Println("Iniciando Buffalo Wings POS Backend...")

	// 1. Cargar configuraciones YAML
	cfg, err := config.Load("config")
	if err != nil {
		log.Fatalf("Error de configuracion: %v", err)
	}
	log.Printf("Configuracion cargada con exito (Entorno: %s, Puerto: %d)", cfg.App.Env, cfg.App.Port)

	// 2. Conectar a MySQL usando variables del entorno / YAML
	dbUser := getEnv("DB_USER", "buffalo_user")
	dbPass := getEnv("DB_PASSWORD", "buffalo_secret_pass")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=UTC",
		dbUser, dbPass, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error abriendo conexion a BD: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	if err := db.Ping(); err != nil {
		log.Fatalf("No se pudo conectar a MySQL: %v", err)
	}
	log.Println("Conexion a MySQL 8.0 establecida correctamente.")

	// 3. Rutas base
	mux := http.NewServeMux()

	// GET /api/v1/healthz
	mux.HandleFunc("GET /api/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		dbStatus := "up"
		if err := db.PingContext(r.Context()); err != nil {
			dbStatus = "down"
			status = http.StatusServiceUnavailable
		}
		httpx.WriteJSON(w, status, map[string]any{
			"status":        "ok",
			"service":       cfg.App.Name,
			"db_status":     dbStatus,
			"timestamp_utc": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// GET /api/v1/config/public
	mux.HandleFunc("GET /api/v1/config/public", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"pin_length":              cfg.Security.PINLength,
			"inactivity_timeout_sec":  cfg.Security.InactivityTimeoutSec,
			"tax_rate":                cfg.Business.TaxRate,
			"suggested_tip_rate":      cfg.Business.SuggestedTipRate,
			"kds_warning_minutes":     cfg.Business.KDSWarningMinutes,
			"kds_critical_minutes":    cfg.Business.KDSCriticalMinutes,
			"enabled_payment_methods": cfg.Business.EnabledPaymentMethods,
		})
	})

	// 4. Autenticación y bitácora (HU-02)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("Falta la variable de entorno JWT_SECRET")
	}

	userRepo := authrepo.NewUserRepository(db)
	auditRepo := auditrepo.NewAuditRepository(db)
	authService, err := authservice.NewAuthService(userRepo, auditRepo, authservice.Config{
		PINLength:         cfg.Security.PINLength,
		MaxFailedAttempts: cfg.Security.MaxFailedAttempts,
		LockoutDuration:   time.Duration(cfg.Security.LockoutMinutes) * time.Minute,
		TokenTTL:          time.Duration(cfg.Security.TokenExpirationMin) * time.Minute,
		JWTSecret:         []byte(jwtSecret),
	})
	if err != nil {
		log.Fatalf("Error creando servicio de autenticacion: %v", err)
	}

	// Límite de intentos de login por IP (login_rate_limit_per_min de security.yaml).
	loginLimiter := ratelimit.New(cfg.Security.LoginRateLimitPerMin, time.Minute)
	authhttp.NewAuthHandler(authService).Register(mux, ratelimit.Middleware(loginLimiter))

	// Mesas (HU-03)
	tableService := tablesservice.NewTableService(tablesrepo.NewTableRepository(db))
	tableshttp.NewTableHandler(tableService).Register(mux, authService)

	// Catálogo y agotados (HU-04)
	catalogService := catalogservice.NewCatalogService(catalogrepo.NewCatalogRepository(db))
	cataloghttp.NewCatalogHandler(catalogService).Register(mux, authService)

	// Comandas (HU-05)
	orderService := ordersservice.NewOrderService(ordersrepo.NewOrderRepository(db))
	ordershttp.NewOrderHandler(orderService).Register(mux, authService)

	// 5. Middlewares globales: CORS envuelve a todo el router.
	handler := httpx.CORS(cfg.CORS.AllowedOrigins)(mux)

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	log.Printf("Servidor escuchando en http://localhost%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Error en servidor HTTP: %v", err)
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
