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
	OrderID    string `json:"委托编号"` // 委托编号
	ContractID string `json:"合同序号"` // 合同序号
}

// OperateAmountView 可操作数量（中文输出）。
type OperateAmountView struct {
	AvailableQuantity string `json:"可操作数量"` // 可操作数量
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

// ToView 将 OperateAmount 转为中文展示。
func (o *OperateAmount) ToView() *OperateAmountView {
	if o == nil {
		return nil
	}
	return &OperateAmountView{
		AvailableQuantity: o.AvailableQuantity,
	}
}

// =============================================================================
// 行情快照展示
// =============================================================================

// FiveQuoteView 五档行情展示。
type FiveQuoteView struct {
	PrevClose      string `json:"昨收"`
	YesSettlePrice string `json:"-"`
	OpenPrice      string `json:"今开"`
	Sale1          string `json:"卖一价"`
	Sale2          string `json:"卖二价"`
	Sale3          string `json:"卖三价"`
	Sale4          string `json:"卖四价"`
	Sale5          string `json:"卖五价"`
	Buy1           string `json:"买一价"`
	Buy2           string `json:"买二价"`
	Buy3           string `json:"买三价"`
	Buy4           string `json:"买四价"`
	Buy5           string `json:"买五价"`
	Sale1Count     int    `json:"卖一量"`
	Sale2Count     int    `json:"卖二量"`
	Sale3Count     int    `json:"卖三量"`
	Sale4Count     int    `json:"卖四量"`
	Sale5Count     int    `json:"卖五量"`
	Buy1Count      int    `json:"买一量"`
	Buy2Count      int    `json:"买二量"`
	Buy3Count      int    `json:"买三量"`
	Buy4Count      int    `json:"买四量"`
	Buy5Count      int    `json:"买五量"`
}

// RealtimeQuoteView 实时行情展示。
type RealtimeQuoteView struct {
	Open         string `json:"今开"`
	High         string `json:"最高"`
	Low          string `json:"最低"`
	Avg          string `json:"均价"`
	ChangeAmount string `json:"涨跌"`
	ChangeRatio  string `json:"涨跌幅"`
	Turnover     string `json:"换手率"`
	CurrentPrice string `json:"最新价"`
	SettlePrice  string `json:"-"`
	Volume       string `json:"成交量"`
	Amount       string `json:"成交额"`
	WP           string `json:"-"`
	NP           string `json:"-"`
	Time         string `json:"时间"`
	Date         string `json:"日期"`
}

// PriceLimitView 涨跌停价展示。
type PriceLimitView struct {
	Upper string `json:"涨停价"`
	Lower string `json:"跌停价"`
}

// SnapshotView 行情快照展示。
type SnapshotView struct {
	Code          string             `json:"股票代码"`
	Name          string             `json:"股票名称"`
	SName         string             `json:"-"`
	Flag          int                `json:"-"`
	TransMarket   int                `json:"-"`
	TransType     int                `json:"-"`
	UpperLimit    string             `json:"涨停价"`
	LowerLimit    string             `json:"跌停价"`
	NT            string             `json:"涨停量"`
	NB            string             `json:"跌停量"`
	Status        int                `json:"-"`
	TradePeriod   int                `json:"-"`
	FiveQuote     *FiveQuoteView     `json:"五档行情"`
	RealtimeQuote *RealtimeQuoteView `json:"实时行情"`
	PriceLimit    *PriceLimitView    `json:"-"`
}

// ToView 将 SnapshotResponse 转为展示结构。
func (s *SnapshotResponse) ToView() *SnapshotView {
	if s == nil {
		return nil
	}
	return &SnapshotView{
		Code:          s.Code,
		Name:          s.Name,
		SName:         s.SName,
		Flag:          s.Flag,
		TransMarket:   s.TransMarket,
		TransType:     s.TransType,
		UpperLimit:    s.UpperLimit,
		LowerLimit:    s.LowerLimit,
		NT:            s.NT,
		NB:            s.NB,
		Status:        s.Status,
		TradePeriod:   s.TradePeriod,
		FiveQuote:     s.FiveQuote.ToView(),
		RealtimeQuote: s.RealtimeQuote.ToView(),
		PriceLimit:    s.PriceLimit.ToView(),
	}
}

// ToView 将 FiveQuote 强转为展示结构。
func (f *FiveQuote) ToView() *FiveQuoteView {
	if f == nil {
		return nil
	}
	v := FiveQuoteView(*f)
	return &v
}

// ToView 将 RealtimeQuote 强转为展示结构。
func (r *RealtimeQuote) ToView() *RealtimeQuoteView {
	if r == nil {
		return nil
	}
	v := RealtimeQuoteView(*r)
	return &v
}

// ToView 将 PriceLimit 强转为展示结构。
func (p *PriceLimit) ToView() *PriceLimitView {
	if p == nil {
		return nil
	}
	v := PriceLimitView(*p)
	return &v
}
