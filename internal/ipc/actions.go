package ipc

const (
	// Request data about the current authentication state
	ACTION_GET_AUTH_DATA             = "getAuthData"
	ACTION_REAUTHENTICATE_SSO        = "reauthenticateSSO"
	ACTION_MUST_REAUTHENTICATE_SSO   = "mustReauthenticateSSO"
	ACTION_FINISH_REAUTHENTICATE_SSO = "finishReauthenticateSSO"
	ACTION_CHANGE_PROFILE            = "changeProfile"

	// Trigger the Tui component to show the error modal
	ACTION_SHOW_ERROR_MODAL = "showErrorModal"

	// Update the error modal with a message
	ACTION_SHOW_ERROR_MESSAGE = "showErrorMessage"

	// Show Reauthhenticate SSO modal
	ACTION_SHOW_REAUTHENTICATE_SSO_MODAL = "showReauthenticateSSOModal"
)

const (
	ACTION_END = "end" // End the application
)

// Component to Component Event Actions
const (
	ACTION_CLOSE_ERROR_MODAL              = "closeErrorModal"
	ACTION_CLOSE_REAUTHENTICATE_SSO_MODAL = "closeReauthenticateSSOModal"
	ACTION_CLOSE_AUTH_MODAL               = "closeAuthModal"
)
