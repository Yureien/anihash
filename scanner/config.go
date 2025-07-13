package scanner

type ScannerConfig struct {
	ScanPath   string `yaml:"scan_path"`
	NumWorkers int    `yaml:"num_workers,omitempty"`
}
