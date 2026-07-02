package client

import (
	"fmt"
	"image"
	"testing"
	"time"
)

// =============================================================================
// Mock Captcha Recognizer
// =============================================================================

// mockRecognizer 模拟验证码识别器。
type mockRecognizer struct {
	result string
	err    error
}

func (m *mockRecognizer) Recognize(img image.Image) (string, error) {
	return m.result, m.err
}

func (m *mockRecognizer) Close() error { return nil }

// =============================================================================
// Mock Logger
// =============================================================================

type mockLogger struct {
	infos  []string
	errors []string
}

func (m *mockLogger) Info(args ...interface{})                   { m.infos = append(m.infos, fmt.Sprint(args...)) }
func (m *mockLogger) Infof(format string, args ...interface{})    { m.infos = append(m.infos, fmt.Sprintf(format, args...)) }
func (m *mockLogger) Debug(args ...interface{})                   {}
func (m *mockLogger) Debugf(format string, args ...interface{})   {}
func (m *mockLogger) Error(args ...interface{})                   { m.errors = append(m.errors, fmt.Sprint(args...)) }
func (m *mockLogger) Errorf(format string, args ...interface{})   { m.errors = append(m.errors, fmt.Sprintf(format, args...)) }

// =============================================================================
// Tests
// =============================================================================

func TestNewClient_NilCaptcha(t *testing.T) {
	c, err := NewClient("user", "pass", nil)
	if err == nil {
		t.Error("expected error for nil captcha")
	}
	if c != nil {
		t.Error("expected nil client")
	}
}

func TestNewClient_Defaults(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}

	c, err := NewClient("testuser", "testpass", mockCap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}

	if c.username != "testuser" {
		t.Errorf("username = %s, want testuser", c.username)
	}
	if c.password != "testpass" {
		t.Errorf("password = %s, want testpass", c.password)
	}
	if c.retryMax != 3 {
		t.Errorf("retryMax = %d, want 3", c.retryMax)
	}
	if c.duration != 30 {
		t.Errorf("duration = %d, want 30", c.duration)
	}
}

func TestNewClient_WithRetry(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap, WithRetry(5, time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.retryMax != 5 {
		t.Errorf("retryMax = %d, want 5", c.retryMax)
	}
	if c.retryWait != time.Second {
		t.Errorf("retryWait = %v, want 1s", c.retryWait)
	}
}

func TestNewClient_WithDuration(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap, WithDuration(60))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.duration != 60 {
		t.Errorf("duration = %d, want 60", c.duration)
	}
}

func TestNewClient_WithLogger(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	log := &mockLogger{}
	c, err := NewClient("user", "pass", mockCap, WithLogger(log))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.logger != log {
		t.Error("logger not set")
	}
}

func TestNewClient_WithLoginFailureCallback(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	called := false
	cb := func(err error) { called = true }
	c, err := NewClient("user", "pass", mockCap, WithLoginFailureCallback(cb))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.onLoginFail == nil {
		t.Error("callback not set")
	}
	_ = called
}

func TestNewClient_SessionCreated(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("testuser", "testpass", mockCap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.session == nil {
		t.Fatal("session is nil")
	}

	// session should return empty key (not logged in yet)
	key, err := c.GetValidateKey()
	if err != nil {
		t.Logf("GetValidateKey error (expected without mock HTTP): %v", err)
	}
	_ = key
}

func TestNoopLogger(t *testing.T) {
	nl := &noopLogger{}
	// 不应 panic
	nl.Info("test")
	nl.Infof("test %d", 1)
	nl.Debug("test")
	nl.Debugf("test %d", 2)
	nl.Error("test")
	nl.Errorf("test %d", 3)
}

func TestDoWithRetry_Success(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap, WithRetry(2, time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	callCount := 0
	data, err := c.doWithRetry(func() ([]byte, error) {
		callCount++
		return []byte("ok"), nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(data) != "ok" {
		t.Errorf("data = %s, want ok", string(data))
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestDoWithRetry_AllFail(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap, WithRetry(2, time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	callCount := 0
	_, err = c.doWithRetry(func() ([]byte, error) {
		callCount++
		return nil, fmt.Errorf("fail")
	})

	if err == nil {
		t.Error("expected error")
	}
	if callCount != 3 { // retryMax + 1
		t.Errorf("callCount = %d, want 3", callCount)
	}
}

func TestDoWithRetry_RetryThenSuccess(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap, WithRetry(2, time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	callCount := 0
	data, err := c.doWithRetry(func() ([]byte, error) {
		callCount++
		if callCount < 3 {
			return nil, fmt.Errorf("fail %d", callCount)
		}
		return []byte("ok"), nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(data) != "ok" {
		t.Errorf("data = %s, want ok", string(data))
	}
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}
}
