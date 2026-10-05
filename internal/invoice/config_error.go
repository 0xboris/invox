package invoice

// ConfigError reports that config.yaml exists but could not be read or parsed.
// Err already names the file, and the line when the YAML parser reports one.
type ConfigError struct {
	Path string
	Err  error
}

func (e *ConfigError) Error() string {
	return e.Err.Error()
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}
