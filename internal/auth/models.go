package auth

import "time"

// User represents a user in the system
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// VerificationCode represents a temporary verification code
type VerificationCode struct {
	Code      string    `json:"code"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RequestCodeRequest is the request payload for requesting a verification code
type RequestCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// VerifyCodeRequest is the request payload for verifying a code
type VerifyCodeRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required,len=6"`
	Username string `json:"username" binding:"min=3,max=50,alphanum"`
}

// AuthResponse is the response after successful authentication
type AuthResponse struct {
	User      *User  `json:"user"`
	SessionID string `json:"session_id"`
}

// UpdateUserRequest is the request payload for updating user information
type UpdateUserRequest struct {
	Username *string `json:"username,omitempty" binding:"omitempty,min=3,max=50,alphanum"`
	Email    *string `json:"email,omitempty" binding:"omitempty,email"`
}

// DeleteUserRequest is the request payload for deleting a user account
type DeleteUserRequest struct {
	Code string `json:"code" binding:"required,len=6"`
}
