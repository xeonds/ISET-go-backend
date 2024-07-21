package lib

type LogConfig struct {
	Level  string `json:"level"`
	Path   string `json:"path"`
	Format string `json:"format"`
}

func InitLog(config LogConfig) error {
	return nil
}
