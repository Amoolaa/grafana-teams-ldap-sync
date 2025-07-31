package grafana

type Config struct {
	URL string `koanf:"url"`

	// env vars
	User     string `koanf:"user"`
	Password string `koanf:"password"`
}
