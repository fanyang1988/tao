package logger

type NullLogger struct {
}

func NewNullLogger() *NullLogger {
	return new(NullLogger)
}

func (n *NullLogger) Tracef(format string, params ...interface{}) {}

func (n *NullLogger) Debugf(format string, params ...interface{}) {}

func (n *NullLogger) Infof(format string, params ...interface{}) {}

func (n *NullLogger) Warnf(format string, params ...interface{}) error {
	return nil
}

func (n *NullLogger) Errorf(format string, params ...interface{}) error {
	return nil
}

func (n *NullLogger) Criticalf(format string, params ...interface{}) error {
	return nil
}

func (n *NullLogger) Trace(v ...interface{}) {}

func (n *NullLogger) Debug(v ...interface{}) {}

func (n *NullLogger) Info(v ...interface{}) {}

func (n *NullLogger) Warn(v ...interface{}) error {
	return nil
}

func (n *NullLogger) Error(v ...interface{}) error {
	return nil
}

func (n *NullLogger) Critical(v ...interface{}) error {
	return nil
}
