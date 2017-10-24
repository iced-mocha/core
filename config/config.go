package config

type Config interface {
	GetStrings(keys []string) ([]string, error)
	GetString(key string) (string, error)
	GetInts(keys []string) ([]int, error)
	GetInt(key string) (int, error)
}
