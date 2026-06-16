package extravars

const (
	BrreweryUser         = "brrewery_user"
	BrreweryGroup        = "brrewery_group"
	BrreweryUserPassword = "brrewery_user_password"
)

const (
	// QbittorrentVersion is the major.minor line chosen in the UI (e.g. "5.2").
	QbittorrentVersion = "qbittorrent_version"
	// QbittorrentRelease is the resolved patch release set by brrewery before Ansible.
	QbittorrentRelease = "qbittorrent_release"
	// LibtorrentBranch selects the libtorrent line (RC_1_2 or RC_2_0) to compile against.
	LibtorrentBranch = "libtorrent_branch"
	// LibtorrentPatch carries an optional, ephemeral base64-encoded libtorrent patch
	// uploaded for a single build. It is never persisted to disk by the API.
	LibtorrentPatch = "libtorrent_patch"
	// QbittorrentQtVersion is the Qt patch release resolved before Ansible runs.
	QbittorrentQtVersion = "qbittorrent_qt_version"
	// QbittorrentZlibVersion is the zlib release resolved before Ansible runs.
	QbittorrentZlibVersion = "qbittorrent_zlib_version"
	// QbittorrentBoostVersion is the Boost release (underscore form, e.g. 1_88_0) resolved before Ansible runs.
	QbittorrentBoostVersion = "qbittorrent_boost_version"
	// QbittorrentOpensslVersion is the OpenSSL 3.x release resolved before Ansible runs.
	QbittorrentOpensslVersion = "qbittorrent_openssl_version"
	// QbittorrentWebUIPasswordHash is the PBKDF2-HMAC-SHA512 password hash in qBittorrent's
	// @ByteArray(<salt_b64>:<hash_b64>) format, computed in Go before the playbook runs.
	QbittorrentWebUIPasswordHash = "qbittorrent_webui_password_hash"
)

const (
	// DelugeVersion is the release line chosen in the UI (e.g. "2.2.x", "1.3.x").
	DelugeVersion = "deluge_version"
	// DelugeRelease is the concrete Deluge version brrewery resolves for the line
	// before Ansible runs (e.g. "2.2.0"); the deluge_build role clones the
	// matching deluge-<release> tag. The chosen libtorrent line is carried by the
	// shared LibtorrentBranch key (RC_1_2/RC_2_0, or RC_1_1 for the 1.3 line).
	DelugeRelease = "deluge_release"
)

const (
	// RtorrentVersion is the release line chosen in the UI (e.g. "0.16.x", "0.9.6").
	RtorrentVersion = "rtorrent_version"
	// RtorrentRelease is the concrete rtorrent version resolved by brrewery before
	// Ansible runs (e.g. "0.16.14"). For ".x" lines it is the newest patch upstream
	// published; for pinned lines it is the manifest tag without the leading "v".
	RtorrentRelease = "rtorrent_release"
	// LibtorrentRelease is the matching rakshasa libtorrent version brrewery resolves
	// for the chosen rtorrent release (e.g. "0.16.14", "0.14.0", "0.13.6"). The rest
	// of the per-line build profile (source mode, C++ standard, rc dialect, patches)
	// is read by the rtorrent_build role from the vendored manifest, not passed here.
	LibtorrentRelease = "libtorrent_release"
)

// BecomePassword is the sudo password used to escalate privileges for app
// operations when brrewery runs unprivileged. The Ansible runner passes it via
// --become-password-file and never places it in the extra-vars JSON or argv.
const BecomePassword = "ansible_become_password" //nolint:gosec // extra-var key name, not a credential

// ForInstall merges caller-supplied vars with the brrewery admin OS user.
// If brrewery_user_password is not explicitly set but the sudo become password
// is present, the become password is reused — they are the same credential.
func ForInstall(username string, extra map[string]string) map[string]string {
	out := make(map[string]string, len(extra)+1)
	for key, value := range extra {
		out[key] = value
	}
	if username != "" {
		out[BrreweryUser] = username
	}
	if out[BrreweryUserPassword] == "" {
		if pw := out[BecomePassword]; pw != "" {
			out[BrreweryUserPassword] = pw
		}
	}
	return out
}
