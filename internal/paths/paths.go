// Package paths defines fixed production paths for brrewery.
package paths

const (
	BinaryPath           = "/usr/local/bin/brrewery"
	BackendListenAddress = "127.0.0.1:8080"
	LogFile              = "/var/log/brrewery/brrewery.log"
	WebRoot              = "/var/www/brrewery"
	UserStorePath        = "/var/lib/brrewery/users.json"
	SessionSecretPath    = "/var/lib/brrewery/session.key" //nolint:gosec // filesystem path, not a secret value
	AnsibleRoot          = "/usr/share/brrewery/ansible"
	NginxSitesAvailable  = "/etc/nginx/sites-available"
	NginxSitesEnabled    = "/etc/nginx/sites-enabled"
	TLSCertPath          = "/etc/ssl/brrewery/fullchain.pem"
	TLSKeyPath           = "/etc/ssl/brrewery/privkey.pem"
)
