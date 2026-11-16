package platform

import (
	"errors"
	"net"
	"os"
	"strings"
)

type Config struct {
	Env            string
	BaseURL        string
	DSN            string
	SMTPAddr       string
	UploadDir      string
	OpsTokenHash   string
	TrustedProxies []string
	OfficeNets     []*net.IPNet
}

func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func Load() (Config, error) {
	cfg := Config{
		Env:            os.Getenv("APP_ENV"),
		BaseURL:        strings.TrimRight(os.Getenv("APP_BASE_URL"), "/"),
		DSN:            os.Getenv("DATABASE_DSN"),
		SMTPAddr:       os.Getenv("SMTP_ADDR"),
		UploadDir:      os.Getenv("UPLOAD_DIR"),
		OpsTokenHash:   strings.ToLower(os.Getenv("OPS_TOKEN_SHA256")),
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
	}
	for _, cidr := range splitList(os.Getenv("OFFICE_NETS")) {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			return cfg, errors.New("OFFICE_NETS: " + err.Error())
		}
		cfg.OfficeNets = append(cfg.OfficeNets, block)
	}
	if cfg.DSN == "" {
		return cfg, errors.New("DATABASE_DSN is required")
	}
	if cfg.Env == "prod" {
		if !strings.HasPrefix(cfg.BaseURL, "https://") {
			return cfg, errors.New("APP_BASE_URL must be https in prod")
		}
		if len(cfg.TrustedProxies) == 0 {
			return cfg, errors.New("TRUSTED_PROXIES is required in prod")
		}
		if !strings.Contains(cfg.DSN, "tls=") {
			return cfg, errors.New("DATABASE_DSN must enable tls in prod")
		}
	}
	return cfg, nil
}

func (c Config) Production() bool { return c.Env == "prod" }

func (c Config) OfficeIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, block := range c.OfficeNets {
		if block.Contains(parsed) {
			return true
		}
	}
	return false
}
