// Package eastmoney 东方财富自动交易接口 Go 语言实现。
package eastmoney

import "time"

// API 端点常量。
const (
	BaseURL = "https://jywg.18.cn"

	// 登录相关
	CaptchaPath  = "/Login/YZM?randNum="
	LoginPath    = "/Login/Authentication?validatekey="
	TradeBuyPage = "/Trade/Buy"

	// 查询接口（validateKey 在运行时拼接）
	QueryAssetAndPos = "/Com/queryAssetAndPositionV1?validatekey="
	QueryOrders      = "/Search/GetOrdersData?validatekey="
	QueryTrades      = "/Search/GetDealData?validatekey="
	QueryHisOrders   = "/Search/GetHisOrdersData?validatekey="
	QueryHisTrades   = "/Search/GetHisDealData?validatekey="
	QueryFundsFlow   = "/Search/GetFundsFlow?validatekey="
	QueryPositions   = "/Search/GetStockList?validatekey="

	// 交易接口
	CreateOrder = "/Trade/SubmitTradeV2?validatekey="
	CancelOrder = "/Trade/RevokeOrders?validatekey="

	// 行情接口（无需登录）
	SnapshotURL = "https://emhsmarketwg.eastmoneysec.com/api/SHSZQuoteSnapshot"
)

// HTTP 请求相关常量。
const (
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"

	// DefaultDuration 登录会话时长（分钟）。
	DefaultDuration = 30

	// DefaultTTLBuffer 会话提前刷新缓冲时间。
	DefaultTTLBuffer = 2 * time.Minute

	// MaxCaptchaRetry 验证码识别最大重试次数。
	MaxCaptchaRetry = 3

	// DefaultRetryMax 默认 HTTP 请求重试次数。
	DefaultRetryMax = 3

	// DefaultRetryWait 默认重试基础等待间隔。
	DefaultRetryWait = 500 * time.Millisecond

	// DefaultHTTPTimeout 默认 HTTP 请求超时。
	DefaultHTTPTimeout = 30 * time.Second
)

// RSAPublicKeyPEM 东方财富密码加密公钥。
const RSAPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDHdsyxT66pDG4p73yope7jxA92
c0AT4qIJ/xtbBcHkFPK77upnsfDTJiVEuQDH+MiMeb+XhCLNKZGp0yaUU6GlxZdp
+nLW8b7Kmijr3iepaDhcbVTsYBWchaWUXauj9Lrhz58/6AE/NF0aMolxIGpsi+ST
2hSHPu3GSXMdhPCkWQIDAQAB
-----END PUBLIC KEY-----`

// URLMap 请求 tag 到 API 端点的映射。
var URLMap = map[string]string{
	"query_asset_and_pos": BaseURL + QueryAssetAndPos,
	"query_orders":        BaseURL + QueryOrders,
	"query_trades":        BaseURL + QueryTrades,
	"query_his_orders":    BaseURL + QueryHisOrders,
	"query_his_trades":    BaseURL + QueryHisTrades,
	"query_funds_flow":    BaseURL + QueryFundsFlow,
	"query_positions":     BaseURL + QueryPositions,
	"create_order":        BaseURL + CreateOrder,
	"cancel_order":        BaseURL + CancelOrder,
}

// BaseHeaders 返回基础请求头的副本。
func BaseHeaders() map[string]string {
	return map[string]string{
		"User-Agent": DefaultUserAgent,
	}
}
