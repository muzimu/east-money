// Package client 提供东方财富 HTTP 客户端，管理会话、查询、交易。
package client

import (
	"encoding/json"
	"fmt"
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

// OperateAmountResponse 可操作数量查询响应。
type OperateAmountResponse struct {
	BaseResponse
	Data []OperateAmount `json:"Data"`
}

// OperateAmount 可操作数量。
type OperateAmount struct {
	AvailableQuantity string `json:"Kczsl"`
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
	OrderDate   string `json:"Wtrq"` // 委托日期
	OrderID     string `json:"Wtbh"` // 委托编号
	StockCode   string `json:"Zqdm"` // 证券代码
	StockName   string `json:"Zqmc"` // 证券名称
	TradeDesc   string `json:"Mmsm"` // 买卖说明
	OrderPrice  string `json:"Wtjg"` // 委托价格
	OrderAmount string `json:"Wtsl"` // 委托数量
	DealAmount  string `json:"Cjsl"` // 成交数量
	OrderStatus string `json:"Wtzt"` // 委托状态
}

// TradesResponse 成交查询响应。
type TradesResponse struct {
	BaseResponse
	Data []TradeRecord `json:"Data"`
}

// TradeRecord 成交记录。
type TradeRecord struct {
	TradeDate  string `json:"Cjrq"` // 成交日期
	TradeID    string `json:"Cjbh"` // 成交编号
	StockCode  string `json:"Zqdm"` // 证券代码
	StockName  string `json:"Zqmc"` // 证券名称
	TradeDesc  string `json:"Mmsm"` // 买卖说明
	TradePrice string `json:"Cjjg"` // 成交价格
	TradeAmt   string `json:"Cjsl"` // 成交数量
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
// 服务端将下单结果置于 Data 数组中，每项含合同序号(Htxh)与委托编号(Wtbh)。
type CreateOrderResponse struct {
	BaseResponse
	Count int               `json:"Count"` // 返回条数
	Data  []CreateOrderData `json:"Data"`  // 下单结果
}

// CreateOrderData 下单返回数据项。
type CreateOrderData struct {
	ContractID string `json:"Htxh"` // 合同序号
	OrderID    string `json:"Wtbh"` // 委托编号
}

// First 返回首条下单结果，无数据时第二个返回值为 false。
func (r *CreateOrderResponse) First() (CreateOrderData, bool) {
	if len(r.Data) > 0 {
		return r.Data[0], true
	}
	return CreateOrderData{}, false
}

// SnapshotResponse 行情快照响应。
type SnapshotResponse struct {
	Code          string         `json:"code"`
	Name          string         `json:"name"`
	SName         string         `json:"sname"`
	Flag          int            `json:"flag"`
	TransMarket   int            `json:"transMarket"`
	TransType     int            `json:"transType"`
	UpperLimit    string         `json:"topprice"`
	LowerLimit    string         `json:"bottomprice"`
	NT            string         `json:"nt"`
	NB            string         `json:"nb"`
	Status        int            `json:"status"`
	TradePeriod   int            `json:"tradeperiod"`
	FiveQuote     *FiveQuote     `json:"fivequote"`
	RealtimeQuote *RealtimeQuote `json:"realtimequote"`
	PriceLimit    *PriceLimit    `json:"pricelimit"`
}

// FiveQuote 五档买卖报价。
type FiveQuote struct {
	PrevClose      string `json:"yesClosePrice"`
	YesSettlePrice string `json:"yesSettlePrice"`
	OpenPrice      string `json:"openPrice"`
	Sale1          string `json:"sale1"`
	Sale2          string `json:"sale2"`
	Sale3          string `json:"sale3"`
	Sale4          string `json:"sale4"`
	Sale5          string `json:"sale5"`
	Buy1           string `json:"buy1"`
	Buy2           string `json:"buy2"`
	Buy3           string `json:"buy3"`
	Buy4           string `json:"buy4"`
	Buy5           string `json:"buy5"`
	Sale1Count     int    `json:"sale1_count"`
	Sale2Count     int    `json:"sale2_count"`
	Sale3Count     int    `json:"sale3_count"`
	Sale4Count     int    `json:"sale4_count"`
	Sale5Count     int    `json:"sale5_count"`
	Buy1Count      int    `json:"buy1_count"`
	Buy2Count      int    `json:"buy2_count"`
	Buy3Count      int    `json:"buy3_count"`
	Buy4Count      int    `json:"buy4_count"`
	Buy5Count      int    `json:"buy5_count"`
}

// RealtimeQuote 实时行情数据。
type RealtimeQuote struct {
	Open         string `json:"open"`
	High         string `json:"high"`
	Low          string `json:"low"`
	Avg          string `json:"avg"`
	ChangeAmount string `json:"zd"`
	ChangeRatio  string `json:"zdf"`
	Turnover     string `json:"turnover"`
	CurrentPrice string `json:"currentPrice"`
	SettlePrice  string `json:"settlePrice"`
	Volume       string `json:"volume"`
	Amount       string `json:"amount"`
	WP           string `json:"wp"`
	NP           string `json:"np"`
	Time         string `json:"time"`
	Date         string `json:"date"`
}

// PriceLimit 涨跌停价格。
type PriceLimit struct {
	Upper string `json:"upper"`
	Lower string `json:"lower"`
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

// String 返回适合日志打印的字符串表示，避免 json.RawMessage 按字节数组输出。
func (r LoginResponse) String() string {
	status := strings.Trim(string(r.Status), `"`)
	if status == "" {
		status = "?"
	}
	return fmt.Sprintf("Status=%s Errcode=%d Message=%s Return_Code=%d", status, r.ErrCode, strings.Trim(r.Message, `"`), r.ReturnCode)
}
