package render

const (
	Ok                  = 0
	Failed              = 1000
	NotFound            = 1001
	ErrParams           = 1002
	TimesUsedUp         = 1003
	ErrForbidden        = 1009
	UserNotExist        = 1052
	NoBindingWallet     = 1062
	RepetitiveOperation = 1068

	ServiceUnavailable = 2001
	ServiceError       = 5001
	UnKnowError        = 6000
)

var statusMsg = map[int]string{
	Ok:                  "Success",
	Failed:              "Failed",
	NotFound:            "Not Found",
	ErrParams:           "Params Error",
	TimesUsedUp:         "Times used up",
	ErrForbidden:        "Forbidden Address",
	ServiceUnavailable:  "Service Unavailable",
	ServiceError:        "Service Error",
	UnKnowError:         "Unknown Error",
	UserNotExist:        "The user account doesn’t exist.",
	NoBindingWallet:     "You have not bound your wallet.",
	RepetitiveOperation: "You repetitive operation.",
}
