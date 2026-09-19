package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL      string
	JWTSecret        string
	APIAddr          string
	PublicBaseURL    string
	CORSOrigin       string
	CookieSecure     bool
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURI  string
	OIDCLabel        string
	OpenAIKey        string
	BlobDir          string
}

func Load() Config {
	loadDotEnv(".env")
	loadDotEnv("../.env")
	return Config{
		DatabaseURL:      env("DATABASE_URL", "postgres://localdiscovery:localdiscovery@localhost:5432/localdiscovery?sslmode=disable"),
		JWTSecret:        env("JWT_SECRET", "dev-jwt-secret-change-me-please-32chars"),
		APIAddr:          env("API_ADDR", ":8080"),
		PublicBaseURL:    env("PUBLIC_BASE_URL", "http://localhost:8080"),
		CORSOrigin:       env("CORS_ORIGIN", "http://localhost:5173"),
		CookieSecure:     envBool("COOKIE_SECURE", false),
		OIDCIssuer:       os.Getenv("OIDC_ISSUER"),
		OIDCClientID:     os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURI:  env("OIDC_REDIRECT_URI", "http://localhost:8080/auth/oidc/callback"),
		OIDCLabel:        env("OIDC_LABEL", "Continue with OpenID"),
		OpenAIKey:        os.Getenv("OPENAI_API_KEY"),
		BlobDir:          env("BLOB_DIR", "./data/blobs"),
	}
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}
