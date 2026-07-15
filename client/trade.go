package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// CreateOrder 提交买入或卖出委托。
//
// 参数：
//   - req: 下单请求参数
func (c *Client) CreateOrder(req *CreateOrderRequest) (*CreateOrderResponse, error) {
	body := url.Values{
		"stockCode": {req.StockCode},
		"tradeType": {req.TradeType},
		"zqmc":      {""},
		"marekt":    {req.Market}, // 服务端字段错误 非拼写错误 请勿纠正
		"price":     {strconv.FormatFloat(req.Price, 'f', -1, 64)},
		"amount":    {strconv.Itoa(req.Amount)},
	}

	data, err := c.querySomething("create_order", body)
	if err != nil {
		return nil, err
	}

	var resp CreateOrderResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析下单响应失败: %w", err)
	}

	c.logger.Infof("下单结果: %v", resp.Data)
	return &resp, nil
}

// CancelOrder 撤销委托。
// 调用 /Trade/cancelStockWEB，需提供委托日期、委托编号、市场与买卖标志四项参数。
// 其中 market 与 mmlb 通常由 QueryRevocableOrders 取得，不建议手工拼凑。
//
// 参数：
//   - req: 撤单请求参数
func (c *Client) CancelOrder(req *CancelOrderRequest) (*CancelOrderResponse, error) {
	body := url.Values{
		"wtrq":   {req.OrderDate},
		"wtbh":   {req.OrderID},
		"market": {req.Market},
		"mmlb":   {req.TradeFlag},
	}

	data, err := c.querySomething("cancel_order", body)
	if err != nil {
		return nil, err
	}

	var resp CancelOrderResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析撤单响应失败: %w", err)
	}

	c.logger.Infof("撤单结果: Status=%d, Message=%s", resp.Status, resp.Message)
	return &resp, nil
}

// CancelOrderByID 按委托标识撤销委托，自动补全撤单所需的市场与买卖标志。
//
// orderStr 支持两种格式：
//   - "委托日期_委托编号"（精确匹配，推荐）
//   - "委托编号"（在当日可撤单中按编号匹配首条）
//
// 撤单接口所需 market 与 mmlb 无法仅凭委托编号推断，
// 故先调用 QueryRevocableOrders 取可撤单列表，匹配出对应委托后取其 Market 与 Mmbz。
func (c *Client) CancelOrderByID(orderStr string) (*CancelOrderResponse, error) {
	wtrq, wtbh, exact := parseOrderID(orderStr)

	revocable, err := c.QueryRevocableOrders()
	if err != nil {
		return nil, fmt.Errorf("查询可撤单失败: %w", err)
	}
	if revocable.Status != 0 {
		return nil, fmt.Errorf("查询可撤单失败: Status=%d, Message=%s", revocable.Status, revocable.Message)
	}

	var matched *RevocableOrder
	for i := range revocable.Data {
		r := &revocable.Data[i]
		if exact {
			if r.OrderDate == wtrq && r.OrderID == wtbh {
				matched = r
				break
			}
		} else if r.OrderID == wtbh {
			matched = r
			break
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("未找到可撤单委托: %s（当日可撤单 %d 笔）", orderStr, len(revocable.Data))
	}

	return c.CancelOrder(&CancelOrderRequest{
		OrderDate: matched.OrderDate,
		OrderID:   matched.OrderID,
		Market:    matched.Market,
		TradeFlag: matched.TradeFlag,
	})
}

// parseOrderID 解析委托标识。
// 含 "_" 视为 "委托日期_委托编号"，返回 exact=true；否则整体作为委托编号。
func parseOrderID(orderStr string) (wtrq, wtbh string, exact bool) {
	if idx := strings.Index(orderStr, "_"); idx > 0 {
		return orderStr[:idx], orderStr[idx+1:], true
	}
	return "", orderStr, false
}
