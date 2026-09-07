package stomp

type Logger interface {
	Debugf(format string, value ...any)
	Infof(format string, value ...any)
	Warningf(format string, value ...any)
	Errorf(format string, value ...any)

	Debug(message string)
	Info(message string)
	Warning(message string)
	Error(message string)
}
