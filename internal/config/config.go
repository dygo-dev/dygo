package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hapyco/dygo/internal/secrets"
	"github.com/hapyco/dygo/internal/shape"
	"github.com/hapyco/dygo/internal/yamlmeta"
	"gopkg.in/yaml.v3"
)

const (
	defaultServerHost     = "127.0.0.1"
	defaultServerPort     = 6790
	defaultDatabaseDriver = "postgres"

	// FilePath is the canonical project-relative dygo runtime config path.
	FilePath = shape.ProjectConfigFile
)

// Config contains dygo runtime settings.
type Config struct {
	Name     string
	Server   Server
	Database Database
	Email    Email
}

// Server contains HTTP server settings.
type Server struct {
	Host string
	Port int
}

// Database contains dygo database settings.
type Database struct {
	Driver string
	URL    SecretReference
}

// Email contains optional framework email delivery settings.
type Email struct {
	From string
	SMTP SMTP
}

// SMTP contains the SMTP transport and encrypted credential references.
type SMTP struct {
	Host     string
	Port     int
	Username SecretReference
	Password SecretReference
}

// Enabled reports whether SMTP email delivery is configured.
func (e Email) Enabled() bool {
	return strings.TrimSpace(e.From) != "" || strings.TrimSpace(e.SMTP.Host) != ""
}

// SecretReference names one encrypted secret value.
type SecretReference struct {
	Secret string
}

// Load reads, decodes, and validates the dygo project config from root.
func Load(root string) (Config, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	path := filepath.Join(root, filepath.FromSlash(FilePath))
	return LoadFile(path)
}

// LoadFile reads, decodes, and validates one dygo config file.
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read dygo config %s: %w", path, err)
	}
	cfg, err := Decode(data)
	if err != nil {
		return Config{}, fmt.Errorf("load dygo config %s: %w", path, err)
	}
	return cfg, nil
}

// Decode decodes and validates one dygo config document.
func Decode(data []byte) (Config, error) {
	if err := rejectDuplicateKeys(data); err != nil {
		return Config{}, err
	}

	cfg := Default()
	var raw rawConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil && err != io.EOF {
		return Config{}, fmt.Errorf("decode dygo config: %w", err)
	}
	raw.apply(&cfg)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Default returns the built-in dygo configuration.
func Default() Config {
	return Config{
		Server: Server{
			Host: defaultServerHost,
			Port: defaultServerPort,
		},
		Database: Database{
			Driver: defaultDatabaseDriver,
		},
	}
}

// Validate checks a resolved dygo config.
func (c Config) Validate() error {
	var problems []string
	if strings.TrimSpace(c.Server.Host) == "" {
		problems = append(problems, "server.host is required")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		problems = append(problems, fmt.Sprintf("server.port must be between 1 and 65535, got %d", c.Server.Port))
	}
	if strings.TrimSpace(c.Database.Driver) == "" {
		problems = append(problems, "database.driver is required")
	} else if c.Database.Driver != defaultDatabaseDriver {
		problems = append(problems, fmt.Sprintf("database.driver must be postgres, got %q", c.Database.Driver))
	}
	if strings.TrimSpace(c.Database.URL.Secret) == "" {
		problems = append(problems, "database.url.secret is required")
	} else if err := secrets.ValidateSecretName(c.Database.URL.Secret); err != nil {
		problems = append(problems, fmt.Sprintf("database.url.secret is invalid: %v", err))
	}
	if c.Email.Enabled() {
		if strings.TrimSpace(c.Email.From) == "" {
			problems = append(problems, "email.from is required when email is configured")
		}
		if strings.TrimSpace(c.Email.SMTP.Host) == "" {
			problems = append(problems, "email.smtp.host is required when email is configured")
		}
		if c.Email.SMTP.Port < 1 || c.Email.SMTP.Port > 65535 {
			problems = append(problems, "email.smtp.port must be between 1 and 65535")
		}
		usernameSecret := strings.TrimSpace(c.Email.SMTP.Username.Secret)
		passwordSecret := strings.TrimSpace(c.Email.SMTP.Password.Secret)
		if (usernameSecret == "") != (passwordSecret == "") {
			problems = append(problems, "email SMTP username and password secrets must be configured together")
		}
		for _, credential := range []struct{ label, name string }{{"username", usernameSecret}, {"password", passwordSecret}} {
			label, name := credential.label, credential.name
			if name != "" {
				if err := secrets.ValidateSecretName(name); err != nil {
					problems = append(problems, fmt.Sprintf("email.smtp.%s.secret is invalid: %v", label, err))
				}
			}
		}
	}
	if len(problems) > 0 {
		return ValidationError{Problems: problems}
	}
	return nil
}

// Address returns the host:port pair used by the server.
func (s Server) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// ValidationError reports one or more config validation problems.
type ValidationError struct {
	Problems []string
}

func (e ValidationError) Error() string {
	return "dygo config validation failed: " + strings.Join(e.Problems, "; ")
}

type rawConfig struct {
	Name     *string      `yaml:"name,omitempty"`
	Server   *rawServer   `yaml:"server,omitempty"`
	Database *rawDatabase `yaml:"database,omitempty"`
	Email    *rawEmail    `yaml:"email,omitempty"`
}

type rawEmail struct {
	From *string  `yaml:"from,omitempty"`
	SMTP *rawSMTP `yaml:"smtp,omitempty"`
}

type rawSMTP struct {
	Host     *string             `yaml:"host,omitempty"`
	Port     *int                `yaml:"port,omitempty"`
	Username *rawSecretReference `yaml:"username,omitempty"`
	Password *rawSecretReference `yaml:"password,omitempty"`
}

type rawServer struct {
	Host *string `yaml:"host,omitempty"`
	Port *int    `yaml:"port,omitempty"`
}

type rawDatabase struct {
	Driver *string             `yaml:"driver,omitempty"`
	URL    *rawSecretReference `yaml:"url,omitempty"`
}

type rawSecretReference struct {
	Secret *string `yaml:"secret,omitempty"`
}

func (r rawConfig) apply(cfg *Config) {
	if r.Name != nil {
		cfg.Name = strings.TrimSpace(*r.Name)
	}
	if r.Server != nil {
		if r.Server.Host != nil {
			cfg.Server.Host = *r.Server.Host
		}
		if r.Server.Port != nil {
			cfg.Server.Port = *r.Server.Port
		}
	}
	if r.Database != nil {
		if r.Database.Driver != nil {
			cfg.Database.Driver = *r.Database.Driver
		}
		if r.Database.URL != nil && r.Database.URL.Secret != nil {
			cfg.Database.URL.Secret = *r.Database.URL.Secret
		}
	}
	if r.Email != nil {
		if r.Email.From != nil {
			cfg.Email.From = strings.TrimSpace(*r.Email.From)
		}
		if r.Email.SMTP != nil {
			if r.Email.SMTP.Host != nil {
				cfg.Email.SMTP.Host = strings.TrimSpace(*r.Email.SMTP.Host)
			}
			if r.Email.SMTP.Port != nil {
				cfg.Email.SMTP.Port = *r.Email.SMTP.Port
			}
			if r.Email.SMTP.Username != nil && r.Email.SMTP.Username.Secret != nil {
				cfg.Email.SMTP.Username.Secret = strings.TrimSpace(*r.Email.SMTP.Username.Secret)
			}
			if r.Email.SMTP.Password != nil && r.Email.SMTP.Password.Secret != nil {
				cfg.Email.SMTP.Password.Secret = strings.TrimSpace(*r.Email.SMTP.Password.Secret)
			}
		}
	}
}

func rejectDuplicateKeys(data []byte) error {
	root, err := yamlmeta.Parse(data, "parse dygo config")
	if err != nil {
		return err
	}
	return yamlmeta.RejectDuplicateKeys(&root, func(duplicate yamlmeta.DuplicateKey) error {
		return fmt.Errorf("duplicate config key %q at line %d, previously defined at line %d", duplicate.Location, duplicate.Line, duplicate.PreviousLine)
	})
}
