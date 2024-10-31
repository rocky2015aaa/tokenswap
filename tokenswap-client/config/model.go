package config

type ConfigRequest struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password"`
}

type UpdateTokenExpirationRequest struct {
	*ConfigRequest
	AccessTokenExpirationTimeInSeconds  int `json:"access_token_expiration_time_in_seconds"`
	RefreshTokenExpirationTimeInSeconds int `json:"refresh_token_expiration_time_in_seconds"`
}

type RefreshRequest struct {
	*ConfigRequest
	RefreshToken string `json:"refresh_token"`
}

type PasswordUpdateRequest struct {
	*ConfigRequest
	NewPassword string `json:"new_password"`
}

type Config struct {
	Email        string `json:"email"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
