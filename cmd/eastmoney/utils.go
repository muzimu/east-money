package main

import (
	"fmt"
	"strconv"
	"strings"
)

// parseTradeArgs 解析交易参数 "CODE-PRICE-AMOUNT"。
func parseTradeArgs(arg string) (code string, price float64, amount int, market string, err error) {
	parts := strings.Split(arg, "-")
	if len(parts) != 3 {
		return "", 0, 0, "", fmt.Errorf("格式错误: 期望 CODE-PRICE-AMOUNT (如 000001-10.50-100)")
	}

	code = parts[0]
	price, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return "", 0, 0, "", fmt.Errorf("无效的价格: %s", parts[1])
	}
	amount, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, 0, "", fmt.Errorf("无效的数量: %s", parts[2])
	}

	market = detectMarket(code)
	return code, price, amount, market, nil
}

// detectMarket 根据股票代码自动判断市场。
// 6xxxxx → HA（上海）, 0xxxxx/3xxxxx → SA（深圳）。
func detectMarket(code string) string {
	if len(code) == 0 {
		return "SA"
	}
	switch code[0] {
	case '6':
		return "HA"
	case '0', '3':
		return "SA"
	default:
		return "SA"
	}
}
