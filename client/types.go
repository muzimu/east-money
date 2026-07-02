// Package client 提供东方财富 HTTP 客户端，管理会话、查询、交易。
package client

import (
	"encoding/json"
	"strings"
)

// =============================================================================
// 请求类型
// =============================================================================

// CreateOrderRequest 下单请求参数。
type CreateOrderRequest struct {
	StockCode string  `json:"stockCode"` // 股票代码
	TradeType string  `json:"tradeType"` // 交易方向: B=买入, S=卖出
	Market    string  `json:"market"`    // 市场: HA=上海, SA=深圳
	Price     float64 `json:"price"`     // 委托价格
	Amount    int     `json:"amount"`    // 委托数量
}

// CancelOrderRequest 撤单请求参数。
type CancelOrderRequest struct {
	Revokes string `json:"revokes"` // 订单标识，格式: "20240520_130662"
}

// HistoryQueryParams 历史查询通用参数。
type HistoryQueryParams struct {
	Size      int    // 查询条数
	StartDate string // 起始日期, "2006-01-02"
	EndDate   string // 结束日期, "2006-01-02"
}

// =============================================================================
// 响应类型（通用）
// =============================================================================

// BaseResponse 所有 API 响应的公共字段。
type BaseResponse struct {
	Status  int    `json:"Status"`
	Message string `json:"Message"`
	Errcode int    `json:"Errcode"`
}

// AssetPositionResponse 资产与持仓查询响应。
type AssetPositionResponse struct {
	BaseResponse
	Data []AccountSummary `json:"Data"`
}

// AccountSummary 账户总览（资金 + 持仓列表）。
type AccountSummary struct {
	TotalAsset  string     `json:"Zzc"`       // 总资产
	MarketValue string     `json:"Zxsz"`      // 证券市值
	Available   string     `json:"Kyzj"`      // 可用资金
	Balance     string     `json:"Zjye"`      // 资金余额
	Frozen      string     `json:"Djzj"`      // 冻结资金
	TotalPL     string     `json:"Ljyk"`      // 累计盈亏
	TodayPL     string     `json:"Dryk"`      // 当日盈亏
	Positions   []Position `json:"positions"` // 持仓明细
}

// Position 持仓记录。
type Position struct {
	StockCode    string `json:"Zqdm"`   // 证券代码
	StockName    string `json:"Zqmc"`   // 证券名称
	FullName     string `json:"zqzwqc"` // 证券全称
	Market       string `json:"Market"` // 市场（SA/HA）
	HoldAmount   string `json:"Zqsl"`   // 持仓数量
	AvailAmount  string `json:"Kysl"`   // 可用数量
	CostPrice    string `json:"Cbjg"`   // 成本价格
	CurrentPrice string `json:"Zxjg"`   // 最新价格
	MarketValue  string `json:"Zxsz"`   // 证券市值
	ProfitLoss   string `json:"Ljyk"`   // 累计盈亏
	TodayPL      string `json:"Dryk"`   // 当日盈亏
	PLRatio      string `json:"Ykbl"`   // 盈亏比例
}

// OrdersResponse 订单查询响应。
type OrdersResponse struct {
	BaseResponse
	Data []OrderRecord `json:"Data"`
}

// OrderRecord 订单记录。
type OrderRecord struct {
	Wtrq       string `json:"Wtrq"` // 委托日期
	Wtbh       string `json:"Wtbh"` // 委托编号
	StockCode  string `json:"Zqdm"` // 证券代码
	StockName  string `json:"Zqmc"` // 证券名称
	TradeType  string `json:"Mmsm"` // 买卖说明
	Price      string `json:"Wtjg"` // 委托价格
	Amount     string `json:"Wtsl"` // 委托数量
	DealAmount string `json:"Cjsl"` // 成交数量
	Status     string `json:"Wtzt"` // 委托状态
	Market     string `json:"Sclx"` // 市场类型
}

// TradesResponse 成交查询响应。
type TradesResponse struct {
	BaseResponse
	Data []TradeRecord `json:"Data"`
}

// TradeRecord 成交记录。
type TradeRecord struct {
	Cjrq      string `json:"Cjrq"` // 成交日期
	Cjbh      string `json:"Cjbh"` // 成交编号
	StockCode string `json:"Zqdm"` // 证券代码
	StockName string `json:"Zqmc"` // 证券名称
	TradeType string `json:"Mmsm"` // 买卖说明
	Price     string `json:"Cjjg"` // 成交价格
	Amount    string `json:"Cjsl"` // 成交数量
	Market    string `json:"Sclx"` // 市场类型
}

// FundsFlowResponse 资金流水查询响应。
type FundsFlowResponse struct {
	BaseResponse
	Data []FundsFlowRecord `json:"Data"`
}

// FundsFlowRecord 资金流水记录。
type FundsFlowRecord struct {
	Date    string `json:"Fsrq"` // 发生日期
	Amount  string `json:"Fsje"` // 发生金额
	Balance string `json:"Zjye"` // 资金余额
	Remark  string `json:"Ywsm"` // 业务说明
}

// CreateOrderResponse 下单响应。
type CreateOrderResponse struct {
	BaseResponse
	Wtrq string `json:"Wtrq"` // 委托日期
	Wtbh string `json:"Wtbh"` // 委托编号
}

// SnapshotResponse 行情快照响应。
type SnapshotResponse struct {
	Status        int            `json:"status"`
	Realtimequote *RealtimeQuote `json:"realtimequote"`
}

// RealtimeQuote 实时行情数据。
type RealtimeQuote struct {
	CurrentPrice string `json:"currentPrice"`
}

// LoginResponse 登录响应。
// Status 字段服务端不一致：成功返回 int 0，失败返回 string "-1"。
type LoginResponse struct {
	Status     json.RawMessage `json:"Status"`
	ErrCode    int             `json:"Errcode"`
	Message    string          `json:"Message"`
	ReturnCode int             `json:"Return_Code"`
}

// IsSuccess 判断登录是否成功（Status 为 0 或 "0"）。
func (r *LoginResponse) IsSuccess() bool {
	s := strings.Trim(string(r.Status), `"`)
	return s == "0"
}
