package config

type Config struct {
	Port    string
	API_KEY string
}

func Load() *Config {
	return &Config{Port: ":8000", API_KEY: ""}
}
