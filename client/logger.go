package client

// noopLogger 空日志实现，默认不输出日志。
type noopLogger struct{}

func (n *noopLogger) Info(args ...any)                  {}
func (n *noopLogger) Infof(format string, args ...any)  {}
func (n *noopLogger) Debug(args ...any)                 {}
func (n *noopLogger) Debugf(format string, args ...any) {}
func (n *noopLogger) Error(args ...any)                 {}
func (n *noopLogger) Errorf(format string, args ...any) {}
