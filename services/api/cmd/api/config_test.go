package main

import "testing"

func TestProductionConfigurationRejectsDevelopmentDefaults(t *testing.T) {
	valid := config{
		appEnv:      "production",
		authSecret:  "different-secret-with-at-least-thirty-two-characters",
		databaseURL: "postgres://app:password@db.example.com:5432/woordenboom?sslmode=verify-full",
		smtpAddr:    "email-smtp.eu-west-1.amazonaws.com:587",
		smtpUser:    "smtp-user",
		smtpPass:    "smtp-password",
		emailFrom:   "login@example.com",
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
		"local mail server":    func(cfg *config) { cfg.smtpAddr = "mailpit:1025" },
		"loopback mail server": func(cfg *config) { cfg.smtpAddr = "127.0.0.1:1025" },
		"missing smtp auth":    func(cfg *config) { cfg.smtpPass = "" },
		"local sender":         func(cfg *config) { cfg.emailFrom = "login@woordenboom.local" },
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
	cfg := config{appEnv: "development", authSecret: "local-development-secret-change-before-production"}
	if err := cfg.validate(); err != nil {
		t.Fatalf("development configuration: %v", err)
	}
}

func TestProductionConfigurationAllowsInternalComposeDatabase(t *testing.T) {
	cfg := config{
		appEnv:      "production",
		authSecret:  "different-secret-with-at-least-thirty-two-characters",
		databaseURL: "postgres://app:password@postgres:5432/woordenboom?sslmode=disable",
		smtpAddr:    "email-smtp.eu-central-1.amazonaws.com:587",
		smtpUser:    "smtp-user",
		smtpPass:    "smtp-password",
		emailFrom:   "login@example.com",
	}
	if err := cfg.validate(); err != nil {
		t.Fatalf("internal Compose database: %v", err)
	}
}
