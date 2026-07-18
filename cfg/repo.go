package cfg

// Repo holds config for the git repo containing the markdown file to sync.
type Repo struct {
	URL              string `env:"REPO_URL,required"`
	Branch           string `env:"REPO_BRANCH,default=main"`
	MarkdownFilePath string `env:"MARKDOWN_FILE_PATH,required"`
	TableID          string `env:"TABLE_ID,required"`
}
