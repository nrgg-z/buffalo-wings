package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	Database DatabaseConfig `yaml:"database"`
	CORS     CORSConfig     `yaml:"cors"`
	Security SecurityConfig `yaml:"security"`
	Business BusinessConfig `yaml:"business"`
}

type AppConfig struct {
	Name             string `yaml:"name"`
	Env              string `yaml:"env"`
	Port             int    `yaml:"port"`
	LogLevel         string `yaml:"log_level"`
	AuditDetailLevel string `yaml:"audit_detail_level"`
}

type DatabaseConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Name         string `yaml:"name"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type SecurityConfig struct {
	PINLength            int `yaml:"pin_length"`
	MaxFailedAttempts    int `yaml:"max_failed_attempts"`
	LockoutMinutes       int `yaml:"lockout_minutes"`
	TokenExpirationMin   int `yaml:"token_expiration_minutes"`
	InactivityTimeoutSec int `yaml:"inactivity_timeout_sec"`
	LoginRateLimitPerMin int `yaml:"login_rate_limit_per_min"`
}

type BusinessConfig struct {
	TaxRate               float64  `yaml:"tax_rate"`
	SuggestedTipRate      float64  `yaml:"suggested_tip_rate"`
	KDSWarningMinutes     int      `yaml:"kds_warning_minutes"`
	KDSCriticalMinutes    int      `yaml:"kds_critical_minutes"`
	DTESimulatedPrefix    string   `yaml:"dte_simulated_prefix"`
	EnabledPaymentMethods []string `yaml:"enabled_payment_methods"`
}

func Load(configDir string) (*Config, error) {
	cfg := &Config{}

	if err := loadYAML(fmt.Sprintf("%s/app.yaml", configDir), cfg); err != nil {
		return nil, fmt.Errorf("error cargando app.yaml: %w", err)
	}
	if err := loadYAML(fmt.Sprintf("%s/security.yaml", configDir), cfg); err != nil {
		return nil, fmt.Errorf("error cargando security.yaml: %w", err)
	}
	if err := loadYAML(fmt.Sprintf("%s/business.yaml", configDir), cfg); err != nil {
		return nil, fmt.Errorf("error cargando business.yaml: %w", err)
	}

	if cfg.Security.PINLength < 4 {
		return nil, fmt.Errorf("seguridad invalida: pin_length debe ser al menos 4")
	}
	if cfg.Security.MaxFailedAttempts < 1 || cfg.Security.LockoutMinutes < 1 {
		return nil, fmt.Errorf("seguridad invalida: max_failed_attempts y lockout_minutes deben ser mayores que 0")
	}
	if cfg.Security.TokenExpirationMin < 1 {
		return nil, fmt.Errorf("seguridad invalida: token_expiration_minutes debe ser mayor que 0")
	}
	if cfg.Security.LoginRateLimitPerMin < 1 {
		return nil, fmt.Errorf("seguridad invalida: login_rate_limit_per_min debe ser mayor que 0")
	}
	if cfg.Business.KDSWarningMinutes >= cfg.Business.KDSCriticalMinutes {
		return nil, fmt.Errorf("negocio invalido: kds_warning_minutes debe ser menor que critical")
	}

	return cfg, nil
}

func loadYAML(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}
