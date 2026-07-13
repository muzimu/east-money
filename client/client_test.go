package client

import (
	"fmt"
	"image"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	eastmoney "github.com/muzimu/east-money"
	"github.com/muzimu/east-money/cache"
	"github.com/stretchr/testify/assert"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type stubSession struct {
	key          string
	forceRelogin int
}

func (s *stubSession) GetValidateKey() (string, error) {
	return s.key, nil
}

func (s *stubSession) ForceReLogin() error {
	s.forceRelogin++
	return nil
}

func (s *stubSession) CookieJar() http.CookieJar {
	jar, _ := cookiejar.New(nil)
	return jar
}

func (m *mockLogger) Info(args ...any) { m.infos = append(m.infos, fmt.Sprint(args...)) }
func (m *mockLogger) Infof(format string, args ...any) {
	m.infos = append(m.infos, fmt.Sprintf(format, args...))
}
func (m *mockLogger) Debug(args ...any)                 {}
func (m *mockLogger) Debugf(format string, args ...any) {}
func (m *mockLogger) Error(args ...any)                 { m.errors = append(m.errors, fmt.Sprint(args...)) }
func (m *mockLogger) Errorf(format string, args ...any) {
	m.errors = append(m.errors, fmt.Sprintf(format, args...))
}

// =============================================================================
// Tests
// =============================================================================

func TestNewClient_NilCaptcha(t *testing.T) {
	c, err := NewClient("user", "pass", nil)
	assert.Error(t, err)
	assert.Nil(t, c)
}

func TestNewClient_Defaults(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}

	c, err := NewClient("testuser", "testpass", mockCap)
	assert.NoError(t, err)
	assert.NotNil(t, c)

	assert.Equal(t, "testuser", c.username)
	assert.Equal(t, "testpass", c.password)
	assert.Equal(t, 3, c.retryMax)
	assert.Equal(t, 30, c.duration)
}

func TestClientNewRequest_AddsBaseHeaders(t *testing.T) {
	c, err := NewClient("testuser", "testpass", &mockRecognizer{result: "1234"})
	assert.NoError(t, err)

	req, err := c.newRequest(http.MethodGet, "https://jywg.18.cn/test", nil)
	assert.NoError(t, err)

	for k, v := range eastmoney.BaseHeaders() {
		assert.Equal(t, v, req.Header.Get(k), "header %s", k)
	}
}

func TestClientNewFormRequest_AddsFormContentType(t *testing.T) {
	c, err := NewClient("testuser", "testpass", &mockRecognizer{result: "1234"})
	assert.NoError(t, err)

	req, err := c.newFormRequest("https://jywg.18.cn/test", url.Values{})
	assert.NoError(t, err)

	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
}

func TestClientNewRequest_DoesNotAddContentType(t *testing.T) {
	c, err := NewClient("testuser", "testpass", &mockRecognizer{result: "1234"})
	assert.NoError(t, err)

	req, err := c.newRequest(http.MethodGet, "https://jywg.18.cn/test", nil)
	assert.NoError(t, err)

	assert.Empty(t, req.Header.Get("Content-Type"))
}

func TestApplyBaseHeaders_DoesNotOverrideExistingHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://jywg.18.cn/test", nil)
	assert.NoError(t, err)
	req.Header.Set("User-Agent", "custom-agent")
	req.Header.Set("Origin", "https://custom.example")
	req.Header.Set("Host", "custom-host.example")

	applyBaseHeaders(req)

	assert.Equal(t, "custom-agent", req.Header.Get("User-Agent"))
	assert.Equal(t, "https://custom.example", req.Header.Get("Origin"))
	assert.Equal(t, "custom-host.example", req.Header.Get("Host"))
}

func TestNewClient_WithRetry(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap, WithRetry(5, time.Second))
	assert.NoError(t, err)
	assert.Equal(t, 5, c.retryMax)
	assert.Equal(t, time.Second, c.retryWait)
}

func TestNewClient_WithDuration(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap, WithDuration(60))
	assert.NoError(t, err)
	assert.Equal(t, 60, c.duration)
}

func TestNewClient_WithLogger(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	log := &mockLogger{}
	c, err := NewClient("user", "pass", mockCap, WithLogger(log))
	assert.NoError(t, err)
	assert.Same(t, log, c.logger)
}

func TestNewClient_WithLoginFailureCallback(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	called := false
	cb := func(err error) { called = true }
	c, err := NewClient("user", "pass", mockCap, WithLoginFailureCallback(cb))
	assert.NoError(t, err)
	assert.NotNil(t, c.onLoginFail)
	_ = called
}

func TestNewClient_WithCaptchaRecognizer(t *testing.T) {
	initial := &mockRecognizer{result: "1234"}
	replacement := &mockRecognizer{result: "5678"}

	c, err := NewClient("user", "pass", initial, WithCaptchaRecognizer(replacement))
	assert.NoError(t, err)

	assert.Same(t, replacement, c.captcha)
}

func TestNewClient_WithHTTPClient(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	customClient := &http.Client{}

	c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
	assert.NoError(t, err)

	assert.Same(t, customClient, c.httpClient)
	assert.NotNil(t, c.httpClient.Jar)
}

func TestClientCookieImportExport(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap)
	assert.NoError(t, err)

	cookies := []*http.Cookie{{Name: "validateKey", Value: "abc123"}}
	c.ImportCookies(cookies)

	exported := c.ExportCookies()

	assert.Len(t, exported, 1)
	assert.Equal(t, "validateKey", exported[0].Name)
	assert.Equal(t, "abc123", exported[0].Value)
	assert.Equal(t, eastmoney.BaseURL, c.baseURL().String())
}

func TestClientSetCacheReplacesSessionJar(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap)
	assert.NoError(t, err)
	oldJar := c.httpClient.Jar

	c.SetCache(cache.NewMemory())

	assert.NotNil(t, c.session)
	assert.NotNil(t, c.httpClient.Jar)
	assert.NotSame(t, oldJar, c.httpClient.Jar)
}

func TestClientForceReLoginReturnsLoginError(t *testing.T) {
	mockCap := &mockRecognizer{result: "not-number"}
	c, err := NewClient("user", "pass", mockCap, WithRetry(1, time.Millisecond))
	assert.NoError(t, err)

	err = c.ForceReLogin()

	assert.Error(t, err)
}

func TestClientExtractValidateKey(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	customClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, eastmoney.DefaultUserAgent, req.Header.Get("User-Agent"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`<input id="em_validatekey" type="hidden" value="validate-123">`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
	assert.NoError(t, err)

	key, err := c.extractValidateKey()

	assert.NoError(t, err)
	assert.Equal(t, "validate-123", key)
}

func TestClientExtractValidateKeyMissing(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	customClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`<html></html>`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
	assert.NoError(t, err)

	key, err := c.extractValidateKey()

	assert.Error(t, err)
	assert.Empty(t, key)
}

func TestClientGetSnapshotAndLastPrice(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	customClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, eastmoney.DefaultUserAgent, req.Header.Get("User-Agent"))
			assert.Equal(t, "600000", req.URL.Query().Get("id"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"Status": 0,
					"Code": "600000",
					"RealtimeQuote": {"CurrentPrice": "12.34"}
				}`)),
				Header: make(http.Header),
			}, nil
		}),
	}
	c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
	assert.NoError(t, err)

	snap, err := c.GetSnapshot("600000")
	assert.NoError(t, err)
	assert.Equal(t, "600000", snap.Code)

	price, err := c.GetLastPrice("600000")
	assert.NoError(t, err)
	assert.Equal(t, 12.34, price)
}

func TestClientGetSnapshotNonOK(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	customClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`error`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
	assert.NoError(t, err)

	snap, err := c.GetSnapshot("600000")

	assert.Error(t, err)
	assert.Nil(t, snap)
}

func TestClientQueryOrders(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	customClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
			assert.Equal(t, "validate-123", req.URL.Query().Get("validatekey"))
			body, err := io.ReadAll(req.Body)
			assert.NoError(t, err)
			form, err := url.ParseQuery(string(body))
			assert.NoError(t, err)
			assert.Equal(t, "100", form.Get("qqhs"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"Status":0,"Data":[{"Wtbh":"order-1"}]}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
	assert.NoError(t, err)
	c.session = &stubSession{key: "validate-123"}

	resp, err := c.QueryOrders()

	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Status)
	assert.Equal(t, []OrderRecord{{OrderID: "order-1"}}, resp.Data)
}

func TestClientQueryOperateAmount(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	customClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
			assert.Equal(t, "validate-123", req.URL.Query().Get("validatekey"))
			body, err := io.ReadAll(req.Body)
			assert.NoError(t, err)
			form, err := url.ParseQuery(string(body))
			assert.NoError(t, err)
			assert.Equal(t, "204001", form.Get("stockCode"))
			assert.Equal(t, "1.415", form.Get("price"))
			assert.Equal(t, "0S", form.Get("tradeType"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"Message":null,"Status":0,"Errcode":0,"Data":[{"Kczsl":"1000"}]}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
	assert.NoError(t, err)
	c.session = &stubSession{key: "validate-123"}

	resp, err := c.QueryOperateAmount("204001", "1.415", "0S")

	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Status)
	assert.Equal(t, []OperateAmount{{AvailableQuantity: "1000"}}, resp.Data)
}

func TestClientQueryMethods(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		call       func(*Client) (any, error)
		assertForm func(*testing.T, url.Values)
		assertResp func(*testing.T, any)
	}{
		{
			name:     "asset and position",
			response: `{"Status":0,"Data":[{"Zzc":"1000","Positions":[{"Zqdm":"600000"}]}]}`,
			call: func(c *Client) (any, error) {
				return c.QueryAssetAndPosition()
			},
			assertForm: func(t *testing.T, form url.Values) {
				assert.Equal(t, "100", form.Get("qqhs"))
			},
			assertResp: func(t *testing.T, got any) {
				resp := got.(*AssetPositionResponse)
				assert.Equal(t, []AccountSummary{{TotalAsset: "1000", Positions: []Position{{StockCode: "600000"}}}}, resp.Data)
			},
		},
		{
			name:     "trades",
			response: `{"Status":0,"Data":[{"Cjbh":"trade-1"}]}`,
			call: func(c *Client) (any, error) {
				return c.QueryTrades()
			},
			assertForm: func(t *testing.T, form url.Values) {
				assert.Equal(t, "100", form.Get("qqhs"))
			},
			assertResp: func(t *testing.T, got any) {
				resp := got.(*TradesResponse)
				assert.Equal(t, []TradeRecord{{TradeID: "trade-1"}}, resp.Data)
			},
		},
		{
			name:     "history orders",
			response: `{"Status":0,"Data":[{"Wtbh":"his-order-1"}]}`,
			call: func(c *Client) (any, error) {
				return c.QueryHistoryOrders(HistoryQueryParams{Size: 20, StartDate: "20260101", EndDate: "20260131"})
			},
			assertForm: assertHistoryForm,
			assertResp: func(t *testing.T, got any) {
				resp := got.(*OrdersResponse)
				assert.Equal(t, []OrderRecord{{OrderID: "his-order-1"}}, resp.Data)
			},
		},
		{
			name:     "history trades",
			response: `{"Status":0,"Data":[{"Cjbh":"his-trade-1"}]}`,
			call: func(c *Client) (any, error) {
				return c.QueryHistoryTrades(HistoryQueryParams{Size: 20, StartDate: "20260101", EndDate: "20260131"})
			},
			assertForm: assertHistoryForm,
			assertResp: func(t *testing.T, got any) {
				resp := got.(*TradesResponse)
				assert.Equal(t, []TradeRecord{{TradeID: "his-trade-1"}}, resp.Data)
			},
		},
		{
			name:     "funds flow",
			response: `{"Status":0,"Data":[{"Fsrq":"20260101","Ywsm":"remark"}]}`,
			call: func(c *Client) (any, error) {
				return c.QueryFundsFlow(HistoryQueryParams{Size: 20, StartDate: "20260101", EndDate: "20260131"})
			},
			assertForm: assertHistoryForm,
			assertResp: func(t *testing.T, got any) {
				resp := got.(*FundsFlowResponse)
				assert.Equal(t, []FundsFlowRecord{{Date: "20260101", Remark: "remark"}}, resp.Data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCap := &mockRecognizer{result: "1234"}
			customClient := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, http.MethodPost, req.Method)
					body, err := io.ReadAll(req.Body)
					assert.NoError(t, err)
					form, err := url.ParseQuery(string(body))
					assert.NoError(t, err)
					tt.assertForm(t, form)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(tt.response)),
						Header:     make(http.Header),
					}, nil
				}),
			}
			c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
			assert.NoError(t, err)
			c.session = &stubSession{key: "validate-123"}

			got, err := tt.call(c)

			assert.NoError(t, err)
			tt.assertResp(t, got)
		})
	}
}

func assertHistoryForm(t *testing.T, form url.Values) {
	assert.Equal(t, "20", form.Get("qqhs"))
	assert.Equal(t, "20260101", form.Get("st"))
	assert.Equal(t, "20260131", form.Get("et"))
}

func TestClientQuerySomethingReloginOnHTML(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	calls := 0
	customClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			body := `{"Status":0,"Data":[]}`
			if calls == 1 {
				body = `<html>expired</html>`
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
	assert.NoError(t, err)
	session := &stubSession{key: "validate-123"}
	c.session = session

	data, err := c.querySomething("query_orders", url.Values{"qqhs": {"100"}})

	assert.NoError(t, err)
	assert.JSONEq(t, `{"Status":0,"Data":[]}`, string(data))
	assert.Equal(t, 1, session.forceRelogin)
	assert.Equal(t, 2, calls)
}

func TestClientQuerySomethingUnknownTag(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap)
	assert.NoError(t, err)
	c.session = &stubSession{key: "validate-123"}

	data, err := c.querySomething("unknown", url.Values{})

	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestClientQuerySomethingHTTPError(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	customClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(`bad request`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
	assert.NoError(t, err)
	c.session = &stubSession{key: "validate-123"}

	data, err := c.querySomething("query_orders", url.Values{})

	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestClientCreateOrder(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	customClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			assert.NoError(t, err)
			form, err := url.ParseQuery(string(body))
			assert.NoError(t, err)
			assert.Equal(t, "600000", form.Get("stockCode"))
			assert.Equal(t, "B", form.Get("tradeType"))
			assert.Equal(t, "1", form.Get("market"))
			assert.Equal(t, "12.34", form.Get("price"))
			assert.Equal(t, "100", form.Get("amount"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"Status":0,"Wtrq":"20260101","Wtbh":"order-1"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
	assert.NoError(t, err)
	c.session = &stubSession{key: "validate-123"}

	resp, err := c.CreateOrder(&CreateOrderRequest{StockCode: "600000", TradeType: "B", Market: "1", Price: 12.34, Amount: 100})

	assert.NoError(t, err)
	assert.Equal(t, "20260101", resp.OrderDate)
	assert.Equal(t, "order-1", resp.OrderID)
}

func TestClientCancelOrder(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	customClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			assert.NoError(t, err)
			form, err := url.ParseQuery(string(body))
			assert.NoError(t, err)
			assert.Equal(t, "order-1", form.Get("revokes"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`cancel-ok`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	c, err := NewClient("user", "pass", mockCap, WithHTTPClient(customClient))
	assert.NoError(t, err)
	c.session = &stubSession{key: "validate-123"}

	result, err := c.CancelOrder("order-1")

	assert.NoError(t, err)
	assert.Equal(t, "cancel-ok", result)
}

func TestNewClient_SessionCreated(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("testuser", "testpass", mockCap)
	assert.NoError(t, err)
	assert.NotNil(t, c.session)

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
	assert.NoError(t, err)

	callCount := 0
	data, err := c.doWithRetry(func() ([]byte, error) {
		callCount++
		return []byte("ok"), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "ok", string(data))
	assert.Equal(t, 1, callCount)
}

func TestDoWithRetry_AllFail(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap, WithRetry(2, time.Millisecond))
	assert.NoError(t, err)

	callCount := 0
	_, err = c.doWithRetry(func() ([]byte, error) {
		callCount++
		return nil, fmt.Errorf("fail")
	})

	assert.Error(t, err)
	assert.Equal(t, 3, callCount) // retryMax + 1
}

func TestDoWithRetry_RetryThenSuccess(t *testing.T) {
	mockCap := &mockRecognizer{result: "1234"}
	c, err := NewClient("user", "pass", mockCap, WithRetry(2, time.Millisecond))
	assert.NoError(t, err)

	callCount := 0
	data, err := c.doWithRetry(func() ([]byte, error) {
		callCount++
		if callCount < 3 {
			return nil, fmt.Errorf("fail %d", callCount)
		}
		return []byte("ok"), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "ok", string(data))
	assert.Equal(t, 3, callCount)
}
