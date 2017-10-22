package config

type Config interface {
	GetString(key string) string
	GetInt(key string) int
}
