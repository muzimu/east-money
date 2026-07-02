package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
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
		"market":    {req.Market},
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

	c.logger.Infof("下单结果: Status=%d, Wtrq=%s, Wtbh=%s", resp.Status, resp.Wtrq, resp.Wtbh)
	return &resp, nil
}

// CancelOrder 撤销委托。
//
// 参数：
//   - orderStr: 订单标识，格式为 "委托日期_委托编号"（如 "20240520_130662"）
func (c *Client) CancelOrder(orderStr string) (string, error) {
	body := url.Values{"revokes": {orderStr}}

	data, err := c.querySomething("cancel_order", body)
	if err != nil {
		return "", err
	}

	result := string(data)
	c.logger.Infof("撤单结果: %s", result)
	return result, nil
}
