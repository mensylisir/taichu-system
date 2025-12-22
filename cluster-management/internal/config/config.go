package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server         ServerConfig         `mapstructure:"server"`
	Database       DatabaseConfig       `mapstructure:"database"`
	Encryption     EncryptionConfig     `mapstructure:"encryption"`
	ClusterManager ClusterManagerConfig `mapstructure:"cluster_manager"`
	Worker         WorkerConfig         `mapstructure:"worker"`
	Logging        LoggingConfig        `mapstructure:"logging"`
	Kubernetes     KubernetesConfig     `mapstructure:"kubernetes"`
}

type ServerConfig struct {
	Port         string        `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	Charset         string        `mapstructure:"charset"`
	Options         string        `mapstructure:"options"`
	Timezone        string        `mapstructure:"timezone"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	AutoMigrate     bool          `mapstructure:"auto_migrate"`
	AutoCreateDB    bool          `mapstructure:"auto_create_db"`
}

type EncryptionConfig struct {
	Key       string `mapstructure:"key"`
	Algorithm string `mapstructure:"algorithm"`
}

type ClusterManagerConfig struct {
	ClientTimeout   time.Duration `mapstructure:"client_timeout"`
	MaxClients      int           `mapstructure:"max_clients"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
}

type WorkerConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	CheckInterval   time.Duration `mapstructure:"check_interval"`
	MaxConcurrency  int           `mapstructure:"max_concurrency"`
	RetryAttempts   int           `mapstructure:"retry_attempts"`
	RetryDelay      time.Duration `mapstructure:"retry_delay"`
	UseInformerMode bool          `mapstructure:"use_informer_mode"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type KubernetesConfig struct {
	QPS     int           `mapstructure:"qps"`
	Burst   int           `mapstructure:"burst"`
	Timeout time.Duration `mapstructure:"timeout"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
