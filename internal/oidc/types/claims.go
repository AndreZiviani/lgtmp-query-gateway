package types

type Claims struct {
	Groups    []string `json:"groups"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
	NotBefore int64    `json:"nbf"`
	Issuer    string   `json:"iss"`
}
