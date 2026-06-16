package auth

// LoginRequest is the body of POST /api/auth/login (architecture §12).
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest is the body of POST /api/auth/register.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// TokenResponse is the login response (architecture §12). It carries no user
// fields, so password hashes can never be serialized through it.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// UserResponse is the safe public view of a user returned by register. It
// deliberately omits PasswordHash.
type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}
