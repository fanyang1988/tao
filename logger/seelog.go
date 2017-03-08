package logger

import "github.com/cihub/seelog"

type SeeLogLogger struct {
}

func NewSeeLogLogger() *SeeLogLogger {
	return new(SeeLogLogger)
}

func (n *SeeLogLogger) Tracef(format string, params ...interface{}) {
	seelog.Tracef(format, params...)
}

func (n *SeeLogLogger) Debugf(format string, params ...interface{}) {
	seelog.Debugf(format, params...)
}

func (n *SeeLogLogger) Infof(format string, params ...interface{}) {
	seelog.Infof(format, params...)
}

func (n *SeeLogLogger) Warnf(format string, params ...interface{}) error {
	return seelog.Warnf(format, params...)
}

func (n *SeeLogLogger) Errorf(format string, params ...interface{}) error {
	return seelog.Errorf(format, params...)
}

func (n *SeeLogLogger) Criticalf(format string, params ...interface{}) error {
	return seelog.Criticalf(format, params...)
}

func (n *SeeLogLogger) Trace(v ...interface{}) {
	seelog.Trace(v...)
}

func (n *SeeLogLogger) Debug(v ...interface{}) {
	seelog.Debug(v...)
}

func (n *SeeLogLogger) Info(v ...interface{}) {
	seelog.Info(v...)
}

func (n *SeeLogLogger) Warn(v ...interface{}) error {
	return seelog.Warn(v...)
}

func (n *SeeLogLogger) Error(v ...interface{}) error {
	return seelog.Error(v...)
}

func (n *SeeLogLogger) Critical(v ...interface{}) error {
	return seelog.Critical(v...)
}
