package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	eastmoney "github.com/muzimu/east-money"
)

// =============================================================================
// 查询接口
// =============================================================================

// QueryAssetAndPosition 查询账户资产与持仓。
func (c *Client) QueryAssetAndPosition() (*AssetPositionResponse, error) {
	body := url.Values{"qqhs": {"100"}, "dwc": {""}}
	data, err := c.querySomething("query_asset_and_pos", body)
	if err != nil {
		return nil, err
	}
	var resp AssetPositionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析资产持仓响应失败: %w", err)
	}
	return &resp, nil
}

// QueryOperateAmount 查询指定证券在价格和交易类型下的可操作数量。
func (c *Client) QueryOperateAmount(stockCode, price, tradeType string) (*OperateAmountResponse, error) {
	body := url.Values{"stockCode": {stockCode}, "price": {price}, "tradeType": {tradeType}}
	data, err := c.querySomething("query_operate_amount", body)
	if err != nil {
		return nil, err
	}
	var resp OperateAmountResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析可操作数量响应失败: %w", err)
	}
	return &resp, nil
}

// QueryOrders 查询当日委托。
func (c *Client) QueryOrders() (*OrdersResponse, error) {
	body := url.Values{"qqhs": {"100"}, "dwc": {""}}
	data, err := c.querySomething("query_orders", body)
	if err != nil {
		return nil, err
	}
	var resp OrdersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析订单响应失败: %w", err)
	}
	return &resp, nil
}

// QueryTrades 查询当日成交。
func (c *Client) QueryTrades() (*TradesResponse, error) {
	body := url.Values{"qqhs": {"100"}, "dwc": {""}}
	data, err := c.querySomething("query_trades", body)
	if err != nil {
		return nil, err
	}
	var resp TradesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析成交响应失败: %w", err)
	}
	return &resp, nil
}

// QueryHistoryOrders 查询历史委托。
//
// 参数：
//   - params: 查询参数（Size, StartDate, EndDate）
func (c *Client) QueryHistoryOrders(params HistoryQueryParams) (*OrdersResponse, error) {
	body := url.Values{
		"qqhs": {strconv.Itoa(params.Size)},
		"dwc":  {""},
		"st":   {params.StartDate},
		"et":   {params.EndDate},
	}
	data, err := c.querySomething("query_his_orders", body)
	if err != nil {
		return nil, err
	}
	var resp OrdersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析历史订单响应失败: %w", err)
	}
	return &resp, nil
}

// QueryHistoryTrades 查询历史成交。
func (c *Client) QueryHistoryTrades(params HistoryQueryParams) (*TradesResponse, error) {
	body := url.Values{
		"qqhs": {strconv.Itoa(params.Size)},
		"dwc":  {""},
		"st":   {params.StartDate},
		"et":   {params.EndDate},
	}
	data, err := c.querySomething("query_his_trades", body)
	if err != nil {
		return nil, err
	}
	var resp TradesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析历史成交响应失败: %w", err)
	}
	return &resp, nil
}

// QueryFundsFlow 查询资金流水。
func (c *Client) QueryFundsFlow(params HistoryQueryParams) (*FundsFlowResponse, error) {
	body := url.Values{
		"qqhs": {strconv.Itoa(params.Size)},
		"dwc":  {""},
		"st":   {params.StartDate},
		"et":   {params.EndDate},
	}
	data, err := c.querySomething("query_funds_flow", body)
	if err != nil {
		return nil, err
	}
	var resp FundsFlowResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析资金流水响应失败: %w", err)
	}
	return &resp, nil
}

// GetLastPrice 获取股票最新价格（无需登录，无需 OCR，可直接调用）。
// 使用默认 HTTP 客户端，适合 CLI 工具等无需复用连接的场景。
func GetLastPrice(symbolCode string) (float64, error) {
	hc := &http.Client{}
	snap, err := getSnapshot(hc, symbolCode)
	if err != nil {
		return 0, err
	}
	return parsePrice(snap)
}

// GetLastPrice 获取股票最新价格（无需登录）。
// 复用 Client 的 HTTP 连接和 Cookie，适合已创建 Client 的场景。
func (c *Client) GetLastPrice(symbolCode string) (float64, error) {
	snap, err := c.GetSnapshot(symbolCode)
	if err != nil {
		return 0, err
	}
	return parsePrice(snap)
}

// GetSnapshot 获取股票完整行情快照（无需登录，无需 OCR，可直接调用）。
// 使用默认 HTTP 客户端，适合 CLI 工具等无需复用连接的场景。
func GetSnapshot(symbolCode string) (*SnapshotResponse, error) {
	hc := &http.Client{}
	return getSnapshot(hc, symbolCode)
}

// GetSnapshot 获取股票完整行情快照（无需登录）。
// 复用 Client 的 HTTP 连接和 Cookie，适合已创建 Client 的场景。
func (c *Client) GetSnapshot(symbolCode string) (*SnapshotResponse, error) {
	return getSnapshot(c.httpClient, symbolCode)
}

// getSnapshot 行情快照查询核心逻辑，通过传入的 http.Client 发送请求。
func getSnapshot(hc *http.Client, symbolCode string) (*SnapshotResponse, error) {
	u := eastmoney.SnapshotURL
	params := url.Values{"id": {symbolCode}}
	fullURL := fmt.Sprintf("%s?%s", u, params.Encode())

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建行情请求失败: %w", err)
	}
	applyBaseHeaders(req)

	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("行情请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("行情返回 HTTP %d", resp.StatusCode)
	}

	var snapResp SnapshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&snapResp); err != nil {
		return nil, fmt.Errorf("解析行情响应失败: %w", err)
	}

	if snapResp.Status != 0 || snapResp.RealtimeQuote == nil {
		return nil, fmt.Errorf("行情数据不可用")
	}

	return &snapResp, nil
}

// parsePrice 从快照响应中提取当前价格。
func parsePrice(snap *SnapshotResponse) (float64, error) {
	price, err := strconv.ParseFloat(snap.RealtimeQuote.CurrentPrice, 64)
	if err != nil {
		return 0, fmt.Errorf("解析价格失败: %w", err)
	}
	return price, nil
}

// =============================================================================
// 内部方法
// =============================================================================

// querySomething 通用查询方法。
// 获取 validateKey → 拼接 URL → POST → 重试。
// 若响应为 HTML（跨进程 cookie 丢失），自动重新登录后重试一次。
func (c *Client) querySomething(tag string, body url.Values) ([]byte, error) {
	return c.querySomethingInner(tag, body, true)
}

func (c *Client) querySomethingInner(tag string, body url.Values, allowRelogin bool) ([]byte, error) {
	key, err := c.session.GetValidateKey()
	if err != nil {
		if c.onLoginFail != nil {
			c.onLoginFail(err)
		}
		return nil, fmt.Errorf("获取 validateKey 失败: %w", err)
	}

	endpoint, ok := eastmoney.URLMap[tag]
	if !ok {
		return nil, fmt.Errorf("未知的请求类型: %s", tag)
	}
	fullURL := endpoint + key

	c.logger.Debugf("请求 %s: %s", tag, fullURL)

	data, err := c.doWithRetry(func() ([]byte, error) {
		req, err := c.newFormRequest(fullURL, body)
		if err != nil {
			return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP 请求失败: %w", err)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取响应失败: %w", err)
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
		}

		return data, nil
	})
	if err != nil {
		return nil, err
	}

	// 跨进程 cookie 丢失：validateKey 命中但服务端无 session → 返回 HTML
	if allowRelogin && len(data) > 0 && data[0] == '<' {
		c.logger.Info("cookie 过期，重新登录...")
		if err := c.session.ForceReLogin(); err != nil {
			return nil, fmt.Errorf("重新登录失败: %w", err)
		}
		return c.querySomethingInner(tag, body, false)
	}

	return data, nil
}

// doWithRetry 带指数退避的重试执行器。
func (c *Client) doWithRetry(fn func() ([]byte, error)) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retryMax; attempt++ {
		if attempt > 0 {
			wait := min(
				// 指数退避
				c.retryWait*time.Duration(1<<(attempt-1)), 30*time.Second)
			c.logger.Debugf("重试 %d/%d, 等待 %v", attempt, c.retryMax, wait)
			time.Sleep(wait)
		}

		data, err := fn()
		if err == nil {
			return data, nil
		}

		lastErr = err
		c.logger.Debugf("请求失败 (尝试 %d/%d): %v", attempt+1, c.retryMax+1, err)
	}
	return nil, fmt.Errorf("全部 %d 次重试均失败: %w", c.retryMax+1, lastErr)
}
