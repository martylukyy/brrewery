package auth

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	TenantID     string `json:"tenant_id"`
}

type UserStore struct {
	Users []User `json:"users"`
}
