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

	market = DetectMarket(code)
	return code, price, amount, market, nil
}

// 市场常量统一管理
const (
	MarketHA = "HA" // 沪市：6xx股票、204逆回购
	MarketSA = "SA" // 深市：0/3股票、1318逆回购
	MarketBJ = "BJ" // 北交所：4/8股票
)

// 前缀映射表：股票 + 逆回购全覆盖
var marketPrefixMap = map[string]string{
	// 股票
	"6": MarketHA,
	"0": MarketSA,
	"3": MarketSA,
	"4": MarketBJ,
	"8": MarketBJ,
	// 沪市逆回购 204开头
	"204": MarketHA,
	// 深市逆回购 1318开头
	"1318": MarketSA,
}

// DetectMarket 兼容普通股票、国债逆回购代码
// code必须6位数字，非法长度默认返回SA
func DetectMarket(code string) string {
	if len(code) != 6 {
		return MarketSA
	}

	// 优先匹配长前缀（逆回购4位 > 3位 > 普通股票1位）
	switch {
	case code[:4] == "1318":
		return MarketSA
	case code[:3] == "204":
		return MarketHA
	}

	// 普通股票1位前缀
	prefix := string(code[0])
	if m, ok := marketPrefixMap[prefix]; ok {
		return m
	}
	return MarketSA
}

// IsReverseRepo 判断是否为国债逆回购
func IsReverseRepo(code string) bool {
	if len(code) != 6 {
		return false
	}
	return code[:3] == "204" || code[:4] == "1318"
}

// parseQueryOperateArgs 解析查询可操作数量参数 "CODE" 或 "CODE-PRICE"。
// 逆回购的价格为可选参数，默认 1。
func parseQueryOperateArgs(arg string) (code string, price string, err error) {
	parts := strings.Split(arg, "-")

	if len(parts) == 1 {
		// 格式: CODE
		code = parts[0]
		if IsReverseRepo(code) {
			price = "1" // 逆回购默认价格
		} else {
			return "", "", fmt.Errorf("普通股票必须指定价格，格式: CODE-PRICE (如 600519-1850)")
		}
	} else if len(parts) == 2 {
		// 格式: CODE-PRICE
		code = parts[0]
		price = parts[1]
		// 验证价格格式
		if _, err := strconv.ParseFloat(price, 64); err != nil {
			return "", "", fmt.Errorf("无效的价格: %s", parts[1])
		}
	} else {
		return "", "", fmt.Errorf("格式错误: 期望 CODE 或 CODE-PRICE (如 204001 或 600519-1850)")
	}

	return code, price, nil
}
