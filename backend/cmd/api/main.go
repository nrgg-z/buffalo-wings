package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"buffalo-pos/internal/auth/adapter/httpapi"
	"buffalo-pos/internal/auth/adapter/mysqlrepo"
	"buffalo-pos/internal/auth/service"
	"buffalo-pos/internal/platform/config"
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
		w.Header().Set("Content-Type", "application/json")
		dbStatus := "up"
		if err := db.Ping(); err != nil {
			dbStatus = "down"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"status":        "ok",
			"service":       cfg.App.Name,
			"db_status":     dbStatus,
			"timestamp_utc": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// GET /api/v1/config/public
	mux.HandleFunc("GET /api/v1/config/public", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"pin_length":              cfg.Security.PINLength,
			"inactivity_timeout_sec":  cfg.Security.InactivityTimeoutSec,
			"tax_rate":                cfg.Business.TaxRate,
			"suggested_tip_rate":      cfg.Business.SuggestedTipRate,
			"kds_warning_minutes":     cfg.Business.KDSWarningMinutes,
			"kds_critical_minutes":    cfg.Business.KDSCriticalMinutes,
			"enabled_payment_methods": cfg.Business.EnabledPaymentMethods,
		})
	})
	// 4. Autenticación (HU-02)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("Falta la variable de entorno JWT_SECRET")
	}

	userRepo := mysqlrepo.NewUserRepository(db)
	authService, err := service.NewAuthService(userRepo, service.Config{
		PINLength:         cfg.Security.PINLength,
		MaxFailedAttempts: cfg.Security.MaxFailedAttempts,
		LockoutDuration:   time.Duration(cfg.Security.LockoutMinutes) * time.Minute,
		TokenTTL:          time.Duration(cfg.Security.TokenExpirationMin) * time.Minute,
		JWTSecret:         []byte(jwtSecret),
	})
	if err != nil {
		log.Fatalf("Error creando servicio de autenticacion: %v", err)
	}
	httpapi.NewAuthHandler(authService).Register(mux)

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	log.Printf("Servidor escuchando en http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Error en servidor HTTP: %v", err)
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
