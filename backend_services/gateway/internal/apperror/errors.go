package apperror

import "net/http"

// AppError is the structured error envelope returned to API clients.
// Every error response follows: {"code": "...", "message": "...", "status": N}
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e *AppError) Error() string { return e.Message }

func New(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

// ── Auth ─────────────────────────────────────────────────────────────

var (
	ErrEmailAlreadyExists       = New("EMAIL_ALREADY_EXISTS", "Email already exists", http.StatusBadRequest)
	ErrInvalidCredentials       = New("INVALID_CREDENTIALS", "Invalid email or password", http.StatusUnauthorized)
	ErrGoogleLoginOnly          = New("GOOGLE_LOGIN_ONLY", "This account uses Google login", http.StatusBadRequest)
	ErrTooManyLoginAttempts     = New("TOO_MANY_LOGIN_ATTEMPTS", "Too many login attempts, please try again later", http.StatusTooManyRequests)
	ErrInvalidRefreshToken      = New("INVALID_REFRESH_TOKEN", "Invalid refresh token", http.StatusUnauthorized)
	ErrCurrentPasswordIncorrect = New("CURRENT_PASSWORD_INCORRECT", "Current password is incorrect", http.StatusBadRequest)
	ErrUserNotFound             = New("USER_NOT_FOUND", "User not found", http.StatusNotFound)
	ErrMissingAuthorization     = New("MISSING_AUTHORIZATION", "Missing authorization", http.StatusUnauthorized)
	ErrInvalidToken             = New("INVALID_TOKEN", "Invalid token", http.StatusUnauthorized)
	ErrInvalidAPIKey            = New("INVALID_API_KEY", "Invalid API key", http.StatusUnauthorized)
)

// ── API Keys ─────────────────────────────────────────────────────────

var (
	ErrInvalidAPIKeyFormat = New("INVALID_API_KEY_FORMAT", "Invalid API key format", http.StatusBadRequest)
)

// ── Organizations ────────────────────────────────────────────────────

var (
	ErrAppNotFound        = New("APP_NOT_FOUND", "App not found", http.StatusNotFound)
	ErrAppNotAvailable    = New("APP_NOT_AVAILABLE", "App is not available", http.StatusForbidden)
	ErrOrgNotFound        = New("ORG_NOT_FOUND", "Organization not found", http.StatusNotFound)
	ErrOrgNameExists      = New("ORG_NAME_EXISTS", "Organization name already exists", http.StatusConflict)
	ErrNotOrgOwner        = New("NOT_ORG_OWNER", "Not organization owner", http.StatusForbidden)
	ErrMemberNotFound     = New("MEMBER_NOT_FOUND", "Member not found", http.StatusNotFound)
	ErrEmailAlreadyMember = New("EMAIL_ALREADY_MEMBER", "User with this email is already a member", http.StatusConflict)
	ErrEmailAlreadyInvited = New("EMAIL_ALREADY_INVITED", "Invite already pending for this email", http.StatusConflict)
	ErrInviteNotFound     = New("INVITE_NOT_FOUND", "Invite not found", http.StatusNotFound)
	ErrInviteNotForUser   = New("INVITE_NOT_FOR_USER", "Invite is not for this user", http.StatusForbidden)
	ErrCannotRemoveOwner  = New("CANNOT_REMOVE_OWNER", "Cannot remove organization owner", http.StatusForbidden)
	ErrCannotUpdateOwner  = New("CANNOT_UPDATE_OWNER", "Cannot update organization owner", http.StatusForbidden)
	ErrInvalidMemberStatus = New("INVALID_MEMBER_STATUS", "Invalid member status", http.StatusBadRequest)
)

// ── Teams ────────────────────────────────────────────────────────────

var (
	ErrTeamNotFound       = New("TEAM_NOT_FOUND", "Team not found", http.StatusNotFound)
	ErrTeamNameExists     = New("TEAM_NAME_EXISTS", "Team name already exists", http.StatusConflict)
	ErrTeamMemberNotFound = New("TEAM_MEMBER_NOT_FOUND", "Team member not found", http.StatusNotFound)
)

// ── Sidebar ──────────────────────────────────────────────────────────

var (
	ErrSidebarNotConfigured = New("SIDEBAR_NOT_CONFIGURED", "Sidebar not configured for app", http.StatusNotFound)
)

// ── Validation ───────────────────────────────────────────────────────

var (
	ErrValidation    = New("VALIDATION_ERROR", "Validation error", http.StatusBadRequest)
	ErrInternalError = New("INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
)

// ValidationError creates a validation AppError with a custom message.
func ValidationError(message string) *AppError {
	return New("VALIDATION_ERROR", message, http.StatusBadRequest)
}
