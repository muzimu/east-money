package main

import (
	"fmt"
	"time"

	"github.com/muzimu/east-money/client"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "登录并显示会话信息（跳过已缓存 Cookie，强制重新认证）",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := createClient(true) // skipCookies=true，支持切换账号
		if err != nil {
			return err
		}

		key, err := c.GetValidateKey()
		if err != nil {
			return fmt.Errorf("登录失败: %w", err)
		}

		printOutput(client.WrapResponse(0, "", 0, &map[string]string{
			"validateKey": key,
		}))
		return nil
	},
}

var buyCmd = &cobra.Command{
	Use:   "buy CODE-PRICE-AMOUNT",
	Short: "买入股票",
	Long: `提交买入委托。格式: 代码-价格-数量

示例:
  eastmoney buy 000001-10.50-100    # 买入平安银行100股，价格10.50
  eastmoney buy 600519-1850.00-100  # 买入贵州茅台100股，价格1850.00`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code, price, amount, market, err := parseTradeArgs(args[0])
		if err != nil {
			return err
		}

		c, err := createClient(false)
		if err != nil {
			return err
		}

		resp, err := c.CreateOrder(&client.CreateOrderRequest{
			StockCode: code,
			TradeType: "B",
			Market:    market,
			Price:     price,
			Amount:    amount,
		})
		if err != nil {
			return err
		}

		view := &client.OrderResultView{
			OrderDate: resp.Wtrq,
			OrderID:   resp.Wtbh,
		}
		printOutput(client.WrapResponse(resp.Status, resp.Message, resp.Errcode, view))
		return nil
	},
}

var sellCmd = &cobra.Command{
	Use:   "sell CODE-PRICE-AMOUNT",
	Short: "卖出股票",
	Long: `提交卖出委托。格式: 代码-价格-数量

示例:
  eastmoney sell 000001-10.50-100    # 卖出平安银行100股，价格10.50`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code, price, amount, market, err := parseTradeArgs(args[0])
		if err != nil {
			return err
		}

		c, err := createClient(false)
		if err != nil {
			return err
		}

		resp, err := c.CreateOrder(&client.CreateOrderRequest{
			StockCode: code,
			TradeType: "S",
			Market:    market,
			Price:     price,
			Amount:    amount,
		})
		if err != nil {
			return err
		}

		view := &client.OrderResultView{
			OrderDate: resp.Wtrq,
			OrderID:   resp.Wtbh,
		}
		printOutput(client.WrapResponse(resp.Status, resp.Message, resp.Errcode, view))
		return nil
	},
}

// queryHandler 查询处理器签名：接收客户端，返回可输出的数据。
type queryHandler func(c *client.Client) (any, error)

// queryHandlers 查询类型 → 处理函数映射，替代大 switch，遵循 OCP。
var queryHandlers = map[string]queryHandler{
	"asset": func(c *client.Client) (any, error) {
		resp, err := c.QueryAssetAndPosition()
		if err != nil {
			return nil, err
		}
		if len(resp.Data) == 0 {
			return client.WrapResponse[client.AccountView](resp.Status, resp.Message, resp.Errcode, nil), nil
		}
		view := resp.Data[0].ToView()
		return client.WrapResponse(resp.Status, resp.Message, resp.Errcode, view), nil
	},
	"order": func(c *client.Client) (any, error) {
		resp, err := c.QueryOrders()
		if err != nil {
			return nil, err
		}
		views := client.ConvertSlice(resp.Data, func(r client.OrderRecord) client.OrderView { return *r.ToView() })
		return client.WrapResponse(resp.Status, resp.Message, resp.Errcode, &views), nil
	},
	"trade": func(c *client.Client) (any, error) {
		resp, err := c.QueryTrades()
		if err != nil {
			return nil, err
		}
		views := client.ConvertSlice(resp.Data, func(r client.TradeRecord) client.TradeView { return *r.ToView() })
		return client.WrapResponse(resp.Status, resp.Message, resp.Errcode, &views), nil
	},
	"history-order": func(c *client.Client) (any, error) {
		resp, err := c.QueryHistoryOrders(client.HistoryQueryParams{
			Size:      flagHistorySize,
			StartDate: flagStartDate,
			EndDate:   flagEndDate,
		})
		if err != nil {
			return nil, err
		}
		views := client.ConvertSlice(resp.Data, func(r client.OrderRecord) client.OrderView { return *r.ToView() })
		return client.WrapResponse(resp.Status, resp.Message, resp.Errcode, &views), nil
	},
	"history-trade": func(c *client.Client) (any, error) {
		resp, err := c.QueryHistoryTrades(client.HistoryQueryParams{
			Size:      flagHistorySize,
			StartDate: flagStartDate,
			EndDate:   flagEndDate,
		})
		if err != nil {
			return nil, err
		}
		views := client.ConvertSlice(resp.Data, func(r client.TradeRecord) client.TradeView { return *r.ToView() })
		return client.WrapResponse(resp.Status, resp.Message, resp.Errcode, &views), nil
	},
	"funds": func(c *client.Client) (any, error) {
		resp, err := c.QueryFundsFlow(client.HistoryQueryParams{
			Size:      flagHistorySize,
			StartDate: flagStartDate,
			EndDate:   flagEndDate,
		})
		if err != nil {
			return nil, err
		}
		views := client.ConvertSlice(resp.Data, func(r client.FundsFlowRecord) client.FundsFlowView { return *r.ToView() })
		return client.WrapResponse(resp.Status, resp.Message, resp.Errcode, &views), nil
	},
}

var queryCmd = &cobra.Command{
	Use:   "query [asset|order|trade|history-order|history-trade|funds]",
	Short: "查询账户信息",
	Long: `查询账户资产、委托、成交或资金流水。

子命令:
  asset         查询资产与持仓
  order         查询当日委托
  trade         查询当日成交
  history-order 查询历史委托（需 --start 和 --end）
  history-trade 查询历史成交（需 --start 和 --end）
  funds         查询资金流水（需 --start 和 --end）`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		handler, ok := queryHandlers[args[0]]
		if !ok {
			return fmt.Errorf("未知查询类型: %s (可选: asset, order, trade, history-order, history-trade, funds)", args[0])
		}

		if err := validateDateRange(args[0]); err != nil {
			return err
		}

		c, err := createClient(false)
		if err != nil {
			return err
		}

		data, err := handler(c)
		if err != nil {
			return err
		}

		printOutput(data)
		return nil
	},
}

var cancelCmd = &cobra.Command{
	Use:   "cancel ORDER_ID",
	Short: "撤销委托",
	Long: `撤销未成交的委托订单。格式: 委托日期_委托编号

示例:
  eastmoney cancel 20240520_130662`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := createClient(false)
		if err != nil {
			return err
		}

		result, err := c.CancelOrder(args[0])
		if err != nil {
			return err
		}

		fmt.Println(result)
		return nil
	},
}

var priceCmd = &cobra.Command{
	Use:   "price CODE [MARKET]",
	Short: "查询股票最新价格",
	Long: `查询股票最新价格。市场默认根据代码自动判断。

示例:
  eastmoney price 000001        # 平安银行（自动识别深圳）
  eastmoney price 600519 HA     # 贵州茅台（手动指定上海）`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		code := args[0]

		price, err := client.GetLastPrice(code)
		if err != nil {
			return err
		}

		fmt.Printf("%s: %.2f\n", code, price)
		return nil
	},
}

// validateDateRange 校验历史查询子命令的日期范围参数。
// history-order / history-trade / funds 需要 --start 和 --end，
// 缺少任一参数时直接报错，不发起无效请求。
func validateDateRange(subcmd string) error {
	switch subcmd {
	case "history-order", "history-trade", "funds":
		if flagStartDate == "" || flagEndDate == "" {
			return fmt.Errorf("%s 需要指定日期范围: --start 和 --end (格式: 2006-01-02)", subcmd)
		}
		if _, err := time.Parse("2006-01-02", flagStartDate); err != nil {
			return fmt.Errorf("无效的起始日期: %s (期望格式: 2006-01-02)", flagStartDate)
		}
		if _, err := time.Parse("2006-01-02", flagEndDate); err != nil {
			return fmt.Errorf("无效的结束日期: %s (期望格式: 2006-01-02)", flagEndDate)
		}
	}
	return nil
}
