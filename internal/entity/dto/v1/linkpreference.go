package dto

// LinkPreferenceRequest represents the request to set link preference on a device.
type LinkPreferenceRequest struct {
	LinkPreference int `json:"linkPreference" binding:"required,min=1,max=2"` // 1 for ME, 2 for HOST
	Timeout        int `json:"timeout" binding:"required,min=0"`              // Timeout in seconds
}

// LinkPreferenceResponse represents the response from setting link preference.
type LinkPreferenceResponse struct {
	ReturnValue    int    `json:"returnValue"`
	ReturnValueStr string `json:"returnValueStr"`
}

// LinkPreference enumeration values
const (
	LinkPreferenceME   = 1 // Management Engine
	LinkPreferenceHost = 2 // Host
)

// Return value constants for SetLinkPreference
const (
	ReturnValueSuccess                    = 0
	ReturnValueNotSupported               = 1
	ReturnValueUnknownFailed              = 2
	ReturnValueTimeout                    = 3
	ReturnValueFailed                     = 4
	ReturnValueInvalidParameter           = 5
	ReturnValueInUse                      = 6
	ReturnValueTransitionStarted          = 4096
	ReturnValueInvalidStateTransition     = 4097
	ReturnValueTimeoutParameterNotSupport = 4098
	ReturnValueBusy                       = 4099
)

// GetReturnValueString returns a human-readable string for the return value.
func GetReturnValueString(returnValue int) string {
	switch returnValue {
	case ReturnValueSuccess:
		return "SUCCESS"
	case ReturnValueNotSupported:
		return "NOT_SUPPORTED"
	case ReturnValueUnknownFailed:
		return "UNKNOWN_FAILED"
	case ReturnValueTimeout:
		return "TIMEOUT"
	case ReturnValueFailed:
		return "FAILED"
	case ReturnValueInvalidParameter:
		return "INVALID_PARAMETER"
	case ReturnValueInUse:
		return "IN_USE"
	case ReturnValueTransitionStarted:
		return "TRANSITION_STARTED"
	case ReturnValueInvalidStateTransition:
		return "INVALID_STATE_TRANSITION"
	case ReturnValueTimeoutParameterNotSupport:
		return "TIMEOUT_PARAMETER_NOT_SUPPORT"
	case ReturnValueBusy:
		return "BUSY"
	default:
		return "UNKNOWN"
	}
}
