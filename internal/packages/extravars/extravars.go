package extravars

const (
	BrreweryUser          = "brrewery_user"
	BrreweryGroup         = "brrewery_group"
	BrreweryUserPassword  = "brrewery_user_password"
)

// ForInstall merges caller-supplied vars with the brrewery admin OS user.
func ForInstall(username string, extra map[string]string) map[string]string {
	out := make(map[string]string, len(extra)+1)
	for key, value := range extra {
		out[key] = value
	}
	if username != "" {
		out[BrreweryUser] = username
	}
	return out
}
