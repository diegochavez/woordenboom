package main

import "testing"

func TestProductionConfigurationRejectsDevelopmentDefaults(t *testing.T) {
	valid := config{
		appEnv:        "production",
		authSecret:    "different-secret-with-at-least-thirty-two-characters",
		databaseURL:   "postgres://app:password@db.example.com:5432/woordenboom?sslmode=verify-full",
		emailProvider: "mailjet",
		mailjetAPIKey: "mailjet-api-key",
		mailjetSecret: "mailjet-secret-key",
		emailFrom:     "login@example.com",
	}
	if err := valid.validate(); err != nil {
		t.Fatalf("valid production configuration: %v", err)
	}
	cases := map[string]func(*config){
		"development auth secret": func(cfg *config) { cfg.authSecret = "local-development-secret-change-before-production" },
		"missing database":        func(cfg *config) { cfg.databaseURL = "" },
		"database without TLS": func(cfg *config) {
			cfg.databaseURL = "postgres://app:password@db.example.com:5432/woordenboom?sslmode=disable"
		},
		"missing Mailjet API key":    func(cfg *config) { cfg.mailjetAPIKey = "" },
		"missing Mailjet secret key": func(cfg *config) { cfg.mailjetSecret = "" },
		"invalid provider":           func(cfg *config) { cfg.emailProvider = "invalid" },
		"local sender":               func(cfg *config) { cfg.emailFrom = "login@woordenboom.local" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			change(&cfg)
			if err := cfg.validate(); err == nil {
				t.Fatal("expected invalid production configuration to be rejected")
			}
		})
	}
}

func TestDevelopmentConfigurationAllowsLocalServices(t *testing.T) {
	cfg := config{appEnv: "development", authSecret: "local-development-secret-change-before-production", emailProvider: "smtp"}
	if err := cfg.validate(); err != nil {
		t.Fatalf("development configuration: %v", err)
	}
}

func TestProductionConfigurationAllowsInternalComposeDatabase(t *testing.T) {
	cfg := config{
		appEnv:        "production",
		authSecret:    "different-secret-with-at-least-thirty-two-characters",
		databaseURL:   "postgres://app:password@postgres:5432/woordenboom?sslmode=disable",
		emailProvider: "mailjet",
		mailjetAPIKey: "mailjet-api-key",
		mailjetSecret: "mailjet-secret-key",
		emailFrom:     "login@example.com",
	}
	if err := cfg.validate(); err != nil {
		t.Fatalf("internal Compose database: %v", err)
	}
}

func TestProductionConfigurationAcceptsExternalSMTP(t *testing.T) {
	cfg := config{
		appEnv:        "production",
		authSecret:    "different-secret-with-at-least-thirty-two-characters",
		databaseURL:   "postgres://app:password@postgres:5432/woordenboom?sslmode=disable",
		emailProvider: "smtp",
		smtpAddr:      "smtp.example.com:587",
		smtpUser:      "smtp-user",
		smtpPass:      "smtp-password",
		emailFrom:     "login@example.com",
	}
	if err := cfg.validate(); err != nil {
		t.Fatalf("external SMTP configuration: %v", err)
	}
}
