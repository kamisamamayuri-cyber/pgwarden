package i18n

// Common UI strings (English).

const (
	ErrInternalServer  = "Internal server error"
	ErrTooManyRequests = "Too many requests"

	ErrInvalidOidcState = "Invalid OIDC state"
	ErrMissingOidcCode  = "Missing OIDC code"
	ErrSsoLoginFailed   = "SSO login failed"

	BtnSave           = "Save"
	BtnDelete         = "Delete"
	BtnCancel         = "Cancel"
	BtnTestConnection = "Test connection"
	BtnShowTasks      = "Show executions"
	BtnShowDetails    = "Details"
	BtnDownload       = "Download"
	BtnCopyClipboard  = "Copy to clipboard"

	LabelStatus           = "Status"
	LabelName             = "Name"
	LabelCreatedAt        = "Created at"
	LabelStartedAt        = "Started at"
	LabelFinishedAt       = "Finished at"
	LabelDuration         = "Duration"
	LabelMessage          = "Message"
	LabelHost             = "Host"
	LabelDatabase         = "Database"
	LabelDestination      = "Destination"
	LabelBackup           = "Backup"
	LabelTask             = "Execution"
	LabelFileSize         = "File size"
	LabelEmail            = "Email"
	LabelYes              = "Yes"
	LabelNo               = "No"
	LabelLocal            = "Local"
	LabelVersion          = "Version"
	LabelConnectionString = "Connection string"
	LabelSchedule         = "Schedule"
	LabelRetention        = "Retention"
	LabelActive           = "Active"
	LabelError            = "Error"
	LabelTestedAt         = "Tested at"
	LabelEndpoint         = "Endpoint"

	MsgConnectionOK        = "Connection successful"
	MsgChartWaitingForData = "No chart data available"
)

func StatusLabel(status string) string {
	switch status {
	case "queued":
		return "Queued"
	case "running":
		return "Running"
	case "success":
		return "Success"
	case "failed":
		return "Failed"
	case "deleted":
		return "Deleted"
	default:
		return status
	}
}
