package server

// JSON response field keys used in API responses to keep the wire format
// consistent and to make repeated string literals explicit constants.
const (
	jsonKeyStatus     = "status"
	jsonKeyVersion    = "version"
	jsonKeyCommit     = "commit"
	jsonKeyBuildDate  = "build_date"
	jsonKeyTimestamp  = "timestamp"
	jsonKeyMessage    = "message"
	jsonKeyError      = "error"
	jsonKeyLimit      = "limit"
	jsonKeyLastMod    = "last_modified"
	jsonKeyTotalFiles = "total_files"
	jsonKeyTotalDirs  = "total_dirs"
	jsonKeyDuration   = "duration"
)

// Common JSON response values.
const (
	jsonStatusSuccess = "success"
	jsonStatusHealthy = "healthy"
	jsonStatusError   = "error"
)

// URL schemes used when constructing absolute URLs.
const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

// HTTP header values used across handlers.
const (
	headerHTTPSProto     = "X-Forwarded-Proto"
	headerForwardedProto = "X-Forwarded-Proto"
	headerContentType    = "Content-Type"
)
