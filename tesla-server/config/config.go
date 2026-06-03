package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Tesla     TeslaConfig
	JWT       JWTConfig
	Map       MapConfig
	AI        AIConfig
	Telemetry TelemetryConfig
}

type TelemetryConfig struct {
	Enabled         bool
	ListenAddr      string
	Hostname        string
	PrivateKey      string
	PublicKeyFile   string
	TLSCertFile     string
	TLSKeyFile      string
	CACertFile      string
	UseDefaultEngCA bool
}

type AIConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type TeslaConfig struct {
	ClientID            string
	ClientSecret        string
	RedirectURI         string
	AuthURL             string
	TokenURL            string
	FleetAPIURL         string
	Audience            string
	VCPURL              string
	FrontendCallbackURL string
	PartnerDomain       string
}

type JWTConfig struct {
	Secret           string
	ExpiresIn        int
	RefreshExpiresIn int
}

type MapConfig struct {
	TencentKey string
}

func Load() *Config {
	teslaTokenURL := getEnv("TESLA_TOKEN_URL", "https://auth.tesla.cn/oauth2/v3/token")
	teslaAuthURL := getEnv("TESLA_AUTH_URL", "https://auth.tesla.cn/oauth2/v3/authorize")
	teslaFleetAPIURL := getEnv("TESLA_FLEET_API_URL", "https://fleet-api.prd.cn.vn.cloud.tesla.cn")
	teslaAudience := getEnv("TESLA_AUDIENCE", "https://fleet-api.prd.cn.vn.cloud.tesla.cn")
	teslaRedirectURI := getEnv("TESLA_REDIRECT_URI", "http://localhost:8080/api/tesla/callback")

	teslaTokenURL = ensureHTTPS(teslaTokenURL)
	teslaAuthURL = ensureHTTPS(teslaAuthURL)
	teslaFleetAPIURL = ensureHTTPS(teslaFleetAPIURL)
	teslaAudience = ensureHTTPS(teslaAudience)
	teslaRedirectURI = ensureHTTPS(teslaRedirectURI)

	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "release"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "tesla_platform"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Tesla: TeslaConfig{
			ClientID:            getEnv("TESLA_CLIENT_ID", ""),
			ClientSecret:        getEnv("TESLA_CLIENT_SECRET", ""),
			RedirectURI:         teslaRedirectURI,
			AuthURL:             teslaAuthURL,
			TokenURL:            teslaTokenURL,
			FleetAPIURL:         teslaFleetAPIURL,
			Audience:            teslaAudience,
			VCPURL:              getEnv("TESLA_VCP_URL", ""),
			FrontendCallbackURL: getEnv("TESLA_FRONTEND_CALLBACK_URL", ""),
			PartnerDomain:       getEnv("TESLA_PARTNER_DOMAIN", ""),
		},
		JWT: JWTConfig{
			Secret:           getEnv("JWT_SECRET", "your-secret-key"),
			ExpiresIn:        getEnvAsInt("JWT_EXPIRES_IN", 7200),
			RefreshExpiresIn: getEnvAsInt("JWT_REFRESH_EXPIRES_IN", 2592000),
		},
		Map: MapConfig{
			TencentKey: getEnv("TENCENT_MAP_KEY", ""),
		},
		AI: AIConfig{
			APIKey:  getEnv("AI_API_KEY", ""),
			Model:   getEnv("AI_MODEL", "glm-4-flash"),
			BaseURL: getEnv("AI_BASE_URL", "https://open.bigmodel.cn/api/paas/v4"),
		},
		Telemetry: TelemetryConfig{
			Enabled:         getEnvAsBool("TELEMETRY_ENABLED", false),
			ListenAddr:      getEnv("TELEMETRY_LISTEN_ADDR", ":8443"),
			Hostname:        getEnv("TELEMETRY_HOSTNAME", ""),
			PrivateKey:      getEnv("TELEMETRY_PRIVATE_KEY", ""),
			PublicKeyFile:   getEnv("TELEMETRY_PUBLIC_KEY", ""),
			TLSCertFile:     getEnv("TELEMETRY_TLS_CERT", ""),
			TLSKeyFile:      getEnv("TELEMETRY_TLS_KEY", ""),
			CACertFile:      getEnv("TELEMETRY_CA_CERT", ""),
			UseDefaultEngCA: getEnvAsBool("TELEMETRY_USE_ENG_CA", false),
		},
	}
}

func ensureHTTPS(url string) string {
	if len(url) > 7 && url[:7] == "http://" {
		return "https://" + url[7:]
	}
	return url
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}
