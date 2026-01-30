package helper

import (
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
)

// Default timeout values for Streamkap resources
const (
	DefaultCreateTimeout = 20 * time.Minute
	DefaultReadTimeout   = 5 * time.Minute
	DefaultUpdateTimeout = 20 * time.Minute
	DefaultDeleteTimeout = 20 * time.Minute
)

// TimeoutsValue is a type alias for timeouts.Value
// Resources can embed this type in their models for timeout support
type TimeoutsValue = timeouts.Value

// TimeoutsType is a type alias for timeouts.Type
// Resources can use this for type conversion
type TimeoutsType = timeouts.Type
