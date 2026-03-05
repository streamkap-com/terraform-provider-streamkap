package constants

// Transform Flink job status values returned by the job_status API.
const (
	JobStatusRunning  = "RUNNING"
	JobStatusFailed   = "FAILED"
	JobStatusCanceled = "CANCELED"
	JobStatusStopped  = "STOPPED"
	JobStatusUnknown  = "UNKNOWN"
)

// DeployVersionLatest is the version ID that tells the backend to use the latest version.
const DeployVersionLatest = "no-version"
