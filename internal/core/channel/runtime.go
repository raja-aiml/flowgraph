package channel

import "time"

// RuntimeConfig controls default behavior for channel constructors.
// Zero values mean "use built-in defaults".
type RuntimeConfig struct {
    InMemoryBufferSize int
    InMemoryTimeout    time.Duration

    BufferedMaxSize int
    BufferedTimeout time.Duration

    PersistentTimeout time.Duration
}

var defaultRuntimeConfig RuntimeConfig

// SetDefaultRuntimeConfig overrides the default channel settings.
func SetDefaultRuntimeConfig(cfg RuntimeConfig) { defaultRuntimeConfig = cfg }

