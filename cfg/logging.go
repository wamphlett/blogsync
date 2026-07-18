package cfg

type Logging struct {
	Level  string `env:"LOG_LEVEL,default=info"`
	Format string `env:"LOG_FORMAT,default=json"`
}
