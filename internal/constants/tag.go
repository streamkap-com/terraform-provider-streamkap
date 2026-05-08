package constants

// DEV_TAG_ID is the system-managed "Development" environment tag. Used as the
// default for streamkap_pipeline.tags so a brand-new pipeline lands in the
// development environment unless the user opts into a different env tag.
const DEV_TAG_ID = "670e5ca40afe1d3983ce0c22"

// TagTypeEnum lists the entity-type values accepted by the backend's
// `app/models/api/app_tags.py::TagTypeEnum`. Mirrored here so the provider can
// validate `streamkap_tag.type` and the `streamkap_tags` data source filter at
// plan time, instead of round-tripping a typo to the backend. Keep in lockstep
// with the backend enum: when a new type ships there, add it here.
//
// Note: `services`, `users`, and `tenant` are reserved for future use upstream
// but are accepted today, so they are included to keep validators forward-
// compatible.
var TagTypeEnum = []string{
	"environment",
	"general",
	"sources",
	"destinations",
	"pipelines",
	"transforms",
	"topics",
	"services",
	"users",
	"tenant",
}
