package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertSlice(t *testing.T) {
	items := []int{1, 2, 3}

	got := ConvertSlice(items, func(item int) string {
		return string(rune('a' + item - 1))
	})

	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestAccountSummaryToView(t *testing.T) {
	var nilAccount *AccountSummary
	assert.Nil(t, nilAccount.ToView())

	account := &AccountSummary{
		TotalAsset:  "1000",
		MarketValue: "800",
		Available:   "200",
		Balance:     "300",
		Frozen:      "10",
		TotalPL:     "20",
		TodayPL:     "5",
		Positions: []Position{
			{StockCode: "600000", StockName: "浦发银行", HoldAmount: "100"},
		},
	}

	view := account.ToView()

	assert.Equal(t, "1000", view.TotalAsset)
	assert.Equal(t, "800", view.MarketValue)
	assert.Equal(t, "200", view.Available)
	assert.Equal(t, "300", view.Balance)
	assert.Equal(t, "10", view.Frozen)
	assert.Equal(t, "20", view.TotalPL)
	assert.Equal(t, "5", view.TodayPL)
	assert.Equal(t, []PositionView{{StockCode: "600000", StockName: "浦发银行", HoldAmount: "100"}}, view.PositionList)
}

func TestSimpleRecordToView(t *testing.T) {
	position := Position{StockCode: "600000", StockName: "浦发银行"}
	assert.Equal(t, PositionView{StockCode: "600000", StockName: "浦发银行"}, position.ToView())

	var nilOrder *OrderRecord
	assert.Nil(t, nilOrder.ToView())
	order := &OrderRecord{OrderDate: "2026-01-01", OrderID: "order-1", StockCode: "600000"}
	assert.Equal(t, &OrderView{OrderDate: "2026-01-01", OrderID: "order-1", StockCode: "600000"}, order.ToView())

	var nilTrade *TradeRecord
	assert.Nil(t, nilTrade.ToView())
	trade := &TradeRecord{TradeDate: "2026-01-01", TradeID: "trade-1", StockCode: "600000"}
	assert.Equal(t, &TradeView{TradeDate: "2026-01-01", TradeID: "trade-1", StockCode: "600000"}, trade.ToView())

	var nilFundsFlow *FundsFlowRecord
	assert.Nil(t, nilFundsFlow.ToView())
	fundsFlow := &FundsFlowRecord{Date: "2026-01-01", Amount: "100", Balance: "1000", Remark: "买入"}
	assert.Equal(t, &FundsFlowView{Date: "2026-01-01", Amount: "100", Balance: "1000", Remark: "买入"}, fundsFlow.ToView())
}

func TestSnapshotToView(t *testing.T) {
	var nilSnapshot *SnapshotResponse
	assert.Nil(t, nilSnapshot.ToView())

	fiveQuote := &FiveQuote{PrevClose: "10.00", Sale1: "10.10", Buy1: "10.00"}
	realtimeQuote := &RealtimeQuote{CurrentPrice: "10.05", Time: "09:30:00", Date: "2026-01-01"}
	priceLimit := &PriceLimit{Upper: "11.00", Lower: "9.00"}
	snapshot := &SnapshotResponse{
		Code:          "600000",
		Name:          "浦发银行",
		SName:         "浦发",
		Flag:          1,
		TransMarket:   2,
		TransType:     3,
		UpperLimit:    "11.00",
		LowerLimit:    "9.00",
		NT:            "nt",
		NB:            "nb",
		Status:        0,
		TradePeriod:   1,
		FiveQuote:     fiveQuote,
		RealtimeQuote: realtimeQuote,
		PriceLimit:    priceLimit,
	}

	view := snapshot.ToView()

	assert.Equal(t, "600000", view.Code)
	assert.Equal(t, "浦发银行", view.Name)
	assert.Equal(t, fiveQuote.ToView(), view.FiveQuote)
	assert.Equal(t, realtimeQuote.ToView(), view.RealtimeQuote)
	assert.Equal(t, priceLimit.ToView(), view.PriceLimit)
}

func TestNestedQuoteToViewNil(t *testing.T) {
	var fiveQuote *FiveQuote
	assert.Nil(t, fiveQuote.ToView())

	var realtimeQuote *RealtimeQuote
	assert.Nil(t, realtimeQuote.ToView())

	var priceLimit *PriceLimit
	assert.Nil(t, priceLimit.ToView())
}
