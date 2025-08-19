package config

type SettingsProvider[T any] interface {
	// GetSettings returns the current settings of type T.
	GetSettings() T
}
