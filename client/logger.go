package client

// noopLogger 空日志实现，默认不输出日志。
type noopLogger struct{}

func (n *noopLogger) Info(args ...interface{})                  {}
func (n *noopLogger) Infof(format string, args ...interface{})  {}
func (n *noopLogger) Debug(args ...interface{})                 {}
func (n *noopLogger) Debugf(format string, args ...interface{}) {}
func (n *noopLogger) Error(args ...interface{})                 {}
func (n *noopLogger) Errorf(format string, args ...interface{}) {}
