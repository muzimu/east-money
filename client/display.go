package client

// =============================================================================
// 输出展示结构体（英文 Go 字段名 + 中文 JSON tag，用于 CLI 美化打印）
// =============================================================================

// AccountView 账户总览（中文输出）。
type AccountView struct {
	TotalAsset   string         `json:"总资产"`  // 总资产
	MarketValue  string         `json:"证券市值"` // 证券市值
	Available    string         `json:"可用资金"` // 可用资金
	Balance      string         `json:"资金余额"` // 资金余额
	Frozen       string         `json:"冻结资金"` // 冻结资金
	TotalPL      string         `json:"累计盈亏"` // 累计盈亏
	TodayPL      string         `json:"当日盈亏"` // 当日盈亏
	PositionList []PositionView `json:"持仓列表"` // 持仓明细
}

// PositionView 持仓记录（中文输出）。
type PositionView struct {
	StockCode    string `json:"证券代码"` // 证券代码
	StockName    string `json:"证券名称"` // 证券名称
	FullName     string `json:"证券全称"` // 证券全称
	HoldAmount   string `json:"持仓数量"` // 持仓数量
	AvailAmount  string `json:"可用数量"` // 可用数量
	CostPrice    string `json:"成本价格"` // 成本价格
	CurrentPrice string `json:"最新价格"` // 最新价格
	MarketValue  string `json:"证券市值"` // 证券市值
	ProfitLoss   string `json:"累计盈亏"` // 累计盈亏
	TodayPL      string `json:"当日盈亏"` // 当日盈亏
	PLRatio      string `json:"盈亏比例"` // 盈亏比例
}

// OrderView 委托记录（中文输出）。
type OrderView struct {
	OrderDate   string `json:"委托日期"` // 委托日期
	OrderID     string `json:"委托编号"` // 委托编号
	StockCode   string `json:"证券代码"` // 证券代码
	StockName   string `json:"证券名称"` // 证券名称
	TradeDesc   string `json:"买卖说明"` // 买卖说明
	OrderPrice  string `json:"委托价格"` // 委托价格
	OrderAmount string `json:"委托数量"` // 委托数量
	DealAmount  string `json:"成交数量"` // 成交数量
	OrderStatus string `json:"委托状态"` // 委托状态
}

// TradeView 成交记录（中文输出）。
type TradeView struct {
	TradeDate  string `json:"成交日期"` // 成交日期
	TradeID    string `json:"成交编号"` // 成交编号
	StockCode  string `json:"证券代码"` // 证券代码
	StockName  string `json:"证券名称"` // 证券名称
	TradeDesc  string `json:"买卖说明"` // 买卖说明
	TradePrice string `json:"成交价格"` // 成交价格
	TradeAmt   string `json:"成交数量"` // 成交数量
}

// FundsFlowView 资金流水记录（中文输出）。
type FundsFlowView struct {
	Date    string `json:"日期"`   // 日期
	Amount  string `json:"发生金额"` // 发生金额
	Balance string `json:"余额"`   // 余额
	Remark  string `json:"备注"`   // 备注
}

// OrderResultView 下单结果（中文输出）。
type OrderResultView struct {
	OrderDate string `json:"委托日期"` // 委托日期
	OrderID   string `json:"委托编号"` // 委托编号
}

// =============================================================================
// 转换函数：API 结构体 → 中文展示结构体
// =============================================================================

// ConvertSlice 泛型批量转换。
func ConvertSlice[S any, V any](items []S, conv func(S) V) []V {
	views := make([]V, len(items))
	for i, item := range items {
		views[i] = conv(item)
	}
	return views
}

// ToView 将 AccountSummary 转为中文展示。
func (a *AccountSummary) ToView() *AccountView {
	if a == nil {
		return nil
	}
	views := ConvertSlice(a.Positions, func(p Position) PositionView { return p.ToView() })
	return &AccountView{
		TotalAsset:   a.TotalAsset,
		MarketValue:  a.MarketValue,
		Available:    a.Available,
		Balance:      a.Balance,
		Frozen:       a.Frozen,
		TotalPL:      a.TotalPL,
		TodayPL:      a.TodayPL,
		PositionList: views,
	}
}

// ToView 将 Position 转为中文展示。
func (p Position) ToView() PositionView { return PositionView(p) }

// ToView 将 OrderRecord 转为中文展示。
func (o *OrderRecord) ToView() *OrderView {
	if o == nil {
		return nil
	}
	v := OrderView(*o)
	return &v
}

// ToView 将 TradeRecord 转为中文展示。
func (t *TradeRecord) ToView() *TradeView {
	if t == nil {
		return nil
	}
	v := TradeView(*t)
	return &v
}

// ToView 将 FundsFlowRecord 转为中文展示。
func (f *FundsFlowRecord) ToView() *FundsFlowView {
	if f == nil {
		return nil
	}
	v := FundsFlowView(*f)
	return &v
}
