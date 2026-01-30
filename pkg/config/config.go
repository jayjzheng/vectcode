package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/jayzheng/vectcode/pkg/embedder"
	"github.com/jayzheng/vectcode/pkg/vectorstore"
)

// Config represents the application configuration
type Config struct {
	VectorStore VectorStoreConfig `yaml:"vector_store"`
	Embeddings  embedder.Config   `yaml:"embeddings"`
	Metadata    MetadataConfig    `yaml:"metadata"`
}

// VectorStoreConfig holds vector store configuration
type VectorStoreConfig struct {
	Type       string            `yaml:"type"`
	Path       string            `yaml:"path"`
	Collection string            `yaml:"collection"`
	Options    map[string]string `yaml:"options"`
}

// MetadataConfig holds metadata store configuration
type MetadataConfig struct {
	DBPath string `yaml:"db_path"`
}

// Load reads and parses the configuration file
func Load(configPath string) (*Config, error) {
	// Expand ~ to home directory
	if configPath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, configPath[2:])
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand ~ in vector store path
	if len(cfg.VectorStore.Path) > 0 && cfg.VectorStore.Path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cfg.VectorStore.Path = filepath.Join(home, cfg.VectorStore.Path[2:])
	}

	// Expand ~ in metadata DB path
	if len(cfg.Metadata.DBPath) > 0 && cfg.Metadata.DBPath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cfg.Metadata.DBPath = filepath.Join(home, cfg.Metadata.DBPath[2:])
	}

	return &cfg, nil
}

// LoadOrDefault loads config from path, or returns default if not found
func LoadOrDefault(configPath string) (*Config, error) {
	cfg, err := Load(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}
	return cfg, nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".vectcode", "db")
	metadataPath := filepath.Join(home, ".vectcode", "metadata.db")

	return &Config{
		VectorStore: VectorStoreConfig{
			Type:       "chroma",
			Path:       dbPath,
			Collection: "vectcode",
		},
		Embeddings: embedder.Config{
			Provider: "ollama",
			Model:    "bge-m3",
			Endpoint: "http://localhost:11434",
		},
		Metadata: MetadataConfig{
			DBPath: metadataPath,
		},
	}
}

// ToVectorStoreConfig converts to vectorstore.Config
func (c *Config) ToVectorStoreConfig() vectorstore.Config {
	return vectorstore.Config{
		Type:       c.VectorStore.Type,
		Path:       c.VectorStore.Path,
		Collection: c.VectorStore.Collection,
		Options:    c.VectorStore.Options,
	}
}
