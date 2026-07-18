package cfg

// Git holds config for authenticating and authoring the commit/push.
type Git struct {
	Token           string `env:"GIT_TOKEN"`
	Remote          string `env:"GIT_REMOTE,default=origin"`
	CommitUserName  string `env:"GIT_COMMIT_USER_NAME,default=blogsync-bot"`
	CommitUserEmail string `env:"GIT_COMMIT_USER_EMAIL,default=blogsync-bot@users.noreply.github.com"`
	CommitMessage   string `env:"GIT_COMMIT_MESSAGE,default=chore: sync blog table"`
}
