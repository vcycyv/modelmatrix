package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Env         string            `yaml:"env"`
	Server      ServerConfig      `yaml:"server"`
	Database    DatabaseConfig    `yaml:"database"`
	LDAP        LDAPConfig        `yaml:"ldap"`
	FileService FileServiceConfig `yaml:"fileservice"`
	JWT         JWTConfig         `yaml:"jwt"`
	Logging     LoggingConfig     `yaml:"logging"`
	Compute     ComputeConfig     `yaml:"compute"`
}

// ServerConfig holds server settings
type ServerConfig struct {
	Port    int    `yaml:"port"`
	Host    string `yaml:"host"`
	BaseURL string `yaml:"base_url"` // External URL for callbacks, e.g., http://localhost:8080
}

// DatabaseConfig holds PostgreSQL connection settings
type DatabaseConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	DBName       string `yaml:"dbname"`
	SSLMode      string `yaml:"sslmode"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

// DSN returns the PostgreSQL connection string
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.Username, d.Password, d.DBName, d.SSLMode,
	)
}

// LDAPConfig holds LDAP connection settings
type LDAPConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	BaseDN       string `yaml:"base_dn"`
	BindDN       string `yaml:"bind_dn"`
	BindPassword string `yaml:"bind_password"`
	UserFilter   string `yaml:"user_filter"`
	GroupFilter  string `yaml:"group_filter"`
	UseTLS       bool   `yaml:"use_tls"`
}

// Address returns the LDAP server address
func (l *LDAPConfig) Address() string {
	return fmt.Sprintf("%s:%d", l.Host, l.Port)
}

// FileServiceConfig holds file storage settings (MinIO)
type FileServiceConfig struct {
	MinioEndpoint  string `yaml:"minio_endpoint"`
	MinioAccessKey string `yaml:"minio_access_key"`
	MinioSecretKey string `yaml:"minio_secret_key"`
	MinioBucket    string `yaml:"minio_bucket"`
	MinioUseSSL    bool   `yaml:"minio_use_ssl"`
}

// JWTConfig holds JWT settings
type JWTConfig struct {
	Secret          string `yaml:"secret"`
	ExpirationHours int    `yaml:"expiration_hours"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level    string `yaml:"level"`
	Format   string `yaml:"format"`
	Output   string `yaml:"output"`
	FilePath string `yaml:"file_path"`
}

// ComputeConfig holds compute service settings
type ComputeConfig struct {
	ServiceURL string `yaml:"service_url"`
	Timeout    int    `yaml:"timeout"` // Timeout in seconds
	APIKey     string `yaml:"api_key"`
}

var cfg *Config

// Load loads configuration from the specified environment
func Load(env string) (*Config, error) {
	if env == "" {
		env = os.Getenv("ENV")
		if env == "" {
			env = "dev"
		}
	}

	configPath := fmt.Sprintf("conf/%s.yaml", env)
	return LoadFromFile(configPath)
}

// LoadFromFile loads configuration from a specific file path
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in the config
	expandedData := expandEnvVars(string(data))

	var config Config
	if err := yaml.Unmarshal([]byte(expandedData), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg = &config
	return cfg, nil
}

// Get returns the current loaded configuration
func Get() *Config {
	return cfg
}

// SetConfig sets the global configuration (primarily for testing)
func SetConfig(c *Config) {
	cfg = c
}

// expandEnvVars replaces ${VAR} with environment variable values
func expandEnvVars(content string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		varName := match[2 : len(match)-1]
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match // Keep original if env var not found
	})
}

// GetEnvInt gets an integer from environment variable with a default value
func GetEnvInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// GetEnvString gets a string from environment variable with a default value
func GetEnvString(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

// GetEnvBool gets a boolean from environment variable with a default value
func GetEnvBool(key string, defaultVal bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultVal
}

