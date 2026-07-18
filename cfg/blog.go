package cfg

// Blog holds config for reaching the blog articles endpoint.
type Blog struct {
	Endpoint string `env:"BLOG_ENDPOINT,required"`
	BaseURL  string `env:"BLOG_BASE_URL,required"`
}
