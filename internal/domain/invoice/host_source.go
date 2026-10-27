package invoice

import "sync"

// HostSource resolves the Host once, on first use, so that the config file
// chosen on the command line counts.
type HostSource struct {
	resolve func(configFile string) Host

	mu         sync.Mutex
	configFile string
	host       *Host
}

// NewHostSource returns a HostSource whose Host is resolve called with the
// config file as typed, "" for the default.
func NewHostSource(resolve func(configFile string) Host) *HostSource {
	return &HostSource{resolve: resolve}
}

// SetConfigFile sets the config file as typed, "" for the default. It must
// be called before the first Host call to count.
func (s *HostSource) SetConfigFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configFile = path
}

// ConfigFile returns the config file as typed, "" when none was set.
func (s *HostSource) ConfigFile() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.configFile
}

// Host returns the user directories, resolved on the first call.
func (s *HostSource) Host() Host {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.host == nil {
		h := s.resolve(s.configFile)
		s.host = &h
	}
	return *s.host
}
