// eastmoney 东方财富自动交易 CLI 工具。
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/muzimu/east-money/cache"
	"github.com/muzimu/east-money/captcha"
	"github.com/muzimu/east-money/client"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Config YAML 配置文件结构。
type Config struct {
	User     string    `yaml:"user"`
	Password string    `yaml:"password"`
	Log      string    `yaml:"log"`
	OCR      OCRConfig `yaml:"ocr"`
}

// OCRConfig OCR 相关配置。
type OCRConfig struct {
	Model   string `yaml:"model"`
	Dict    string `yaml:"dict"`
	ONNXLib string `yaml:"onnx_lib"`
}

// 全局变量：CLI flags 值（优先级最高）。
var (
	flagUser     string
	flagPassword string
	flagModel    string
	flagDict     string
	flagONNXLib  string
	flagConfig   string
	flagSession  string
	flagLog      string

	cmdLogger     zerolog.Logger
	currentClient *client.Client

	// 合并后的最终配置
	cfg = Config{
		Log: ".eastmoney/eastmoney.log",
		OCR: OCRConfig{
			Model: "./go-ocr/ddddocr_weights/common.onnx",
			Dict:  "./go-ocr/ddddocr_weights/dict.txt",
		},
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "eastmoney",
	Short: "东方财富自动交易 CLI",
	Long: `East Money CLI — 东方财富自动交易命令行工具。

支持登录、查询（资产/订单/成交）、买入、卖出、撤单等操作。
验证码使用本地 ddddocr ONNX 模型识别。

配置文件: ./config.yaml`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := mergeConfig(); err != nil {
			return err
		}
		initLogger()
		cmdLogger.Info().Str("args", fmt.Sprint(args)).Msg(cmd.CommandPath())
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		saveCookies()
		cmdLogger.Info().Msg(cmd.CommandPath() + " 完成")
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagUser, "user", "u", "", "账户用户名（环境变量: EM_USERNAME）")
	rootCmd.PersistentFlags().StringVarP(&flagPassword, "pass", "p", "", "账户密码（环境变量: EM_PASSWORD）")
	rootCmd.PersistentFlags().StringVar(&flagModel, "model", "", "ddddocr 模型文件路径")
	rootCmd.PersistentFlags().StringVar(&flagDict, "dict", "", "ddddocr 字典文件路径")
	rootCmd.PersistentFlags().StringVar(&flagONNXLib, "onnx-lib", "", "onnxruntime 共享库路径（留空自动检测 go-ocr/lib/）")
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "", "配置文件路径（默认 ./config.yaml）")
	rootCmd.PersistentFlags().StringVar(&flagSession, "session", ".eastmoney/session.json", "会话持久化文件路径")
	rootCmd.PersistentFlags().StringVar(&flagLog, "log", ".eastmoney/eastmoney.log", "日志文件路径")

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(buyCmd)
	rootCmd.AddCommand(sellCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(cancelCmd)
	rootCmd.AddCommand(priceCmd)
}

// mergeConfig 按优先级合并配置：CLI flags > 环境变量 > YAML 配置 > 默认值。
func mergeConfig() error {
	// 1. 加载 YAML 配置文件
	configPath := resolveConfigPath()
	if data, err := os.ReadFile(configPath); err == nil {
		var fileCfg Config
		if err := yaml.Unmarshal(data, &fileCfg); err != nil {
			return fmt.Errorf("解析配置文件 %s 失败: %w", configPath, err)
		}
		applyFileConfig(&fileCfg)
	}

	// 2. 环境变量覆盖
	if v := os.Getenv("EM_USERNAME"); v != "" {
		cfg.User = v
	}
	if v := os.Getenv("EM_PASSWORD"); v != "" {
		cfg.Password = v
	}

	if flagLog != "" {
		cfg.Log = flagLog
	}
	// 3. CLI flags 覆盖（最高优先级）
	if flagUser != "" {
		cfg.User = flagUser
	}
	if flagPassword != "" {
		cfg.Password = flagPassword
	}
	if flagModel != "" {
		cfg.OCR.Model = flagModel
	}
	if flagDict != "" {
		cfg.OCR.Dict = flagDict
	}
	if flagONNXLib != "" {
		cfg.OCR.ONNXLib = flagONNXLib
	}

	// 4. ONNX 库自动检测（仅当未显式指定时）
	if cfg.OCR.ONNXLib == "" {
		cfg.OCR.ONNXLib = autoDetectONNXLib()
	}

	return nil
}

// applyFileConfig 将配置文件中的非零值写入全局配置。
func applyFileConfig(fc *Config) {
	if fc.User != "" {
		cfg.User = fc.User
	}
	if fc.Password != "" {
		cfg.Password = fc.Password
	}
	if fc.Log != "" {
		cfg.Log = fc.Log
	}
	if fc.OCR.Model != "" {
		cfg.OCR.Model = fc.OCR.Model
	}
	if fc.OCR.Dict != "" {
		cfg.OCR.Dict = fc.OCR.Dict
	}
	if fc.OCR.ONNXLib != "" {
		cfg.OCR.ONNXLib = fc.OCR.ONNXLib
	}
}

// resolveConfigPath 确定配置文件路径。
// 优先级: --config flag > ./config.yaml
func resolveConfigPath() string {
	if flagConfig != "" {
		return flagConfig
	}
	return "config.yaml"
}

// autoDetectONNXLib 自动检测 ONNX Runtime 库路径。
// 搜索顺序：./go-ocr/lib/ → ../go-ocr/lib/ → 系统路径（brew / apt） → 返回空
func autoDetectONNXLib() string {
	var libName string
	var systemPaths []string
	switch runtime.GOOS {
	case "darwin":
		libName = fmt.Sprintf("onnxruntime_%s.dylib", runtime.GOARCH)
		systemPaths = []string{
			"/opt/homebrew/lib/libonnxruntime.dylib",
			"/usr/local/lib/libonnxruntime.dylib",
		}
	case "linux":
		libName = fmt.Sprintf("onnxruntime_%s.so", runtime.GOARCH)
		systemPaths = []string{
			"/usr/lib/libonnxruntime.so",
			"/usr/local/lib/libonnxruntime.so",
		}
	case "windows":
		libName = "onnxruntime.dll"
	default:
		return ""
	}

	// 系统路径优先（brew/apt 版本通常更新）
	searchPaths := systemPaths
	searchPaths = append(searchPaths,
		filepath.Join("go-ocr", "lib", libName),
		filepath.Join("..", "go-ocr", "lib", libName),
	)

	for _, p := range searchPaths {
		if info, err := os.Stat(p); err == nil && info.Size() > 1024 {
			return p
		}
	}

	return ""
}

// resolveCredentials 从合并后的配置获取凭据。
func resolveCredentials() (string, string) {
	return cfg.User, cfg.Password
}

// zerologAdapter 将 zerolog.Logger 适配为 client.Logger 接口。
type zerologAdapter struct{ log zerolog.Logger }

func (z *zerologAdapter) Info(args ...any)             { z.log.Info().Msg(fmt.Sprint(args...)) }
func (z *zerologAdapter) Infof(f string, args ...any)  { z.log.Info().Msgf(f, args...) }
func (z *zerologAdapter) Debug(args ...any)            { z.log.Debug().Msg(fmt.Sprint(args...)) }
func (z *zerologAdapter) Debugf(f string, args ...any) { z.log.Debug().Msgf(f, args...) }
func (z *zerologAdapter) Error(args ...any)            { z.log.Error().Msg(fmt.Sprint(args...)) }
func (z *zerologAdapter) Errorf(f string, args ...any) { z.log.Error().Msgf(f, args...) }

// initLogger 初始化文件日志。
func initLogger() {
	if dir := filepath.Dir(cfg.Log); dir != "." {
		os.MkdirAll(dir, 0700)
	}
	logFile, err := os.OpenFile(cfg.Log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建日志文件失败: %v\n", err)
		cmdLogger = zerolog.New(os.Stderr).With().Timestamp().Logger()
		return
	}
	cmdLogger = zerolog.New(logFile).With().Timestamp().Logger()
}

const cookieFile = ".eastmoney/cookies.json"

func saveCookies() {
	if currentClient == nil {
		return
	}
	cookies := currentClient.ExportCookies()
	data, _ := json.Marshal(cookies)
	os.WriteFile(cookieFile, data, 0600)
}

func loadCookies(c *client.Client) {
	data, err := os.ReadFile(cookieFile)
	if err != nil {
		return
	}
	var cookies []*http.Cookie
	if json.Unmarshal(data, &cookies) != nil {
		return
	}
	c.ImportCookies(cookies)
}

// createClient 创建交易客户端。
// 使用文件缓存持久化登录状态，跨 CLI 调用复用 validateKey。
func createClient() (*client.Client, error) {
	u, p := resolveCredentials()
	if u == "" || p == "" {
		return nil, fmt.Errorf("请提供用户名和密码（-u/-p、环境变量或配置文件）")
	}

	recognizer, err := captcha.NewDefaultRecognizer(cfg.OCR.Model, cfg.OCR.Dict, cfg.OCR.ONNXLib)
	if err != nil {
		return nil, fmt.Errorf("创建 OCR 引擎失败: %w", err)
	}

	c, err := client.NewClient(u, p, recognizer, client.WithLogger(&zerologAdapter{cmdLogger}))
	if err != nil {
		recognizer.Close()
		return nil, fmt.Errorf("创建客户端失败: %w", err)
	}

	// 会话持久化（validateKey + cookies）
	if dir := filepath.Dir(flagSession); dir != "." {
		os.MkdirAll(dir, 0700)
	}
	c.SetCache(cache.NewFile(flagSession))
	loadCookies(c)

	currentClient = c
	return c, nil
}

// =============================================================================
// 子命令
// =============================================================================

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "登录并显示会话信息",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := createClient()
		if err != nil {
			return err
		}

		key, err := c.GetValidateKey()
		if err != nil {
			return fmt.Errorf("登录失败: %w", err)
		}

		fmt.Printf("登录成功！\n")
		fmt.Printf("ValidateKey: %s\n", key)
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

		c, err := createClient()
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

		printJSON(resp)
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

		c, err := createClient()
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

		printJSON(resp)
		return nil
	},
}

var queryCmd = &cobra.Command{
	Use:   "query [asset|order|trade]",
	Short: "查询账户信息",
	Long: `查询账户资产、委托或成交。

子命令:
  asset  查询资产与持仓
  order  查询当日委托
  trade  查询当日成交`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := createClient()
		if err != nil {
			return err
		}

		switch args[0] {
		case "asset":
			resp, err := c.QueryAssetAndPosition()
			if err != nil {
				return err
			}
			printJSON(resp)
		case "order":
			resp, err := c.QueryOrders()
			if err != nil {
				return err
			}
			printJSON(resp)
		case "trade":
			resp, err := c.QueryTrades()
			if err != nil {
				return err
			}
			printJSON(resp)
		default:
			return fmt.Errorf("未知查询类型: %s (可选: asset, order, trade)", args[0])
		}
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
		c, err := createClient()
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

// =============================================================================
// 辅助函数
// =============================================================================

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

func printJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("%+v\n", v)
		return
	}
	fmt.Println(string(b))
}
