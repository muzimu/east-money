// eastmoney 东方财富自动交易 CLI 工具。
package main

import (
	"fmt"
	"os"

	"github.com/muzimu/east-money/client"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// 全局变量：CLI flags 值（优先级最高）。
var (
	flagUser      string
	flagPassword  string
	flagModel     string
	flagDict      string
	flagONNXLib   string
	flagOCRRemote string
	flagConfig    string
	flagSession   string
	flagLog       string

	// 历史查询参数
	flagHistorySize    int
	flagStartDate      string
	flagEndDate        string

	// 内部使用的查询参数（不暴露给用户）
	flagQueryStockCode string
	flagQueryPrice     string
	flagQueryTradeType string

	// 输出格式
	flagFormat string

	cmdLogger     zerolog.Logger
	currentClient *client.Client

	// 合并后的最终配置
	cfg = Config{
		Log: ".eastmoney/eastmoney.log",
		OCR: OCRConfig{
			Model: "./go-ocr-model/ddddocr/common.onnx",
			Dict:  "./go-ocr-model/ddddocr/dict.txt",
		},
	}

	// 版本信息由 GoReleaser 或 Makefile 通过 -ldflags 注入。
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// flag/参数解析错误时显示用法说明
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(cmd.UsageString())
		return err
	})

	if err := rootCmd.Execute(); err != nil {
		// 业务错误通过 printOutput 按 --format 格式输出
		flagFormat = resolveFormat()
		printOutput(map[string]string{
			"状态": "失败",
			"错误": err.Error(),
		})
		os.Exit(1)
	}
}

// resolveFormat 返回当前生效的输出格式，优先使用 flagFormat；
// 当 flagFormat 未初始化（如 PersistentPreRunE 之前出错）时回退到 human。
func resolveFormat() string {
	if flagFormat != "" {
		return flagFormat
	}
	return formatHuman
}

var rootCmd = &cobra.Command{
	Use:           "eastmoney",
	Short:         "东方财富自动交易 CLI",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `East Money CLI — 东方财富自动交易命令行工具。

支持登录、查询（资产/订单/成交/历史）、买入、卖出、撤单等操作。
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
	// 持久化 flags（所有子命令可用）
	rootCmd.PersistentFlags().StringVarP(&flagUser, "user", "u", "", "账户用户名（环境变量: EM_USERNAME）")
	rootCmd.PersistentFlags().StringVarP(&flagPassword, "pass", "p", "", "账户密码（环境变量: EM_PASSWORD）")
	rootCmd.PersistentFlags().StringVar(&flagModel, "model", "", "ddddocr 模型文件路径")
	rootCmd.PersistentFlags().StringVar(&flagDict, "dict", "", "ddddocr 字典文件路径")
	rootCmd.PersistentFlags().StringVar(&flagONNXLib, "onnx-lib", "", "onnxruntime 共享库路径（留空自动检测 go-ocr/lib/）")
	rootCmd.PersistentFlags().StringVar(&flagOCRRemote, "ocr-remote", "", "远程 OCR 服务地址（如 http://localhost:8000/ocr），设置后跳过本地 ONNX 模型")
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "./config.yaml", "配置文件路径")
	rootCmd.PersistentFlags().StringVar(&flagSession, "session", ".eastmoney/session.json", "会话持久化文件路径")
	rootCmd.PersistentFlags().StringVar(&flagLog, "log", ".eastmoney/eastmoney.log", "日志文件路径")
	rootCmd.PersistentFlags().StringVar(&flagFormat, "format", "human", "输出格式: json | human")

	// 历史查询 flags
	rootCmd.PersistentFlags().IntVar(&flagHistorySize, "size", 20, "历史查询条数")
	rootCmd.PersistentFlags().StringVar(&flagStartDate, "start", "", "查询起始日期 (2006-01-02)")
	rootCmd.PersistentFlags().StringVar(&flagEndDate, "end", "", "查询结束日期 (2006-01-02)")

	// 可操作数量查询 flags（内部使用，通过位置参数传入）
	rootCmd.PersistentFlags().StringVar(&flagQueryStockCode, "query-stock-code", "", "")
	rootCmd.PersistentFlags().StringVar(&flagQueryPrice, "query-price", "", "")
	rootCmd.PersistentFlags().StringVar(&flagQueryTradeType, "query-trade-type", "", "")
	rootCmd.PersistentFlags().MarkHidden("query-stock-code")
	rootCmd.PersistentFlags().MarkHidden("query-price")
	rootCmd.PersistentFlags().MarkHidden("query-trade-type")

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(buyCmd)
	rootCmd.AddCommand(sellCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(cancelCmd)
	rootCmd.AddCommand(priceCmd)
	rootCmd.AddCommand(versionCmd)
}
