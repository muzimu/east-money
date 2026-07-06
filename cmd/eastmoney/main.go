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
	flagUser     string
	flagPassword string
	flagModel    string
	flagDict     string
	flagONNXLib  string
	flagConfig   string
	flagSession  string
	flagLog      string

	// 历史查询参数
	flagHistorySize int
	flagStartDate   string
	flagEndDate     string

	// 输出格式
	flagFormat string

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

	// 版本信息由 GoReleaser 或 Makefile 通过 -ldflags 注入。
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
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
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "./config.yaml", "配置文件路径")
	rootCmd.PersistentFlags().StringVar(&flagSession, "session", ".eastmoney/session.json", "会话持久化文件路径")
	rootCmd.PersistentFlags().StringVar(&flagLog, "log", ".eastmoney/eastmoney.log", "日志文件路径")
	rootCmd.PersistentFlags().StringVar(&flagFormat, "format", "human", "输出格式: json | human")

	// 历史查询 flags
	rootCmd.PersistentFlags().IntVar(&flagHistorySize, "size", 20, "历史查询条数")
	rootCmd.PersistentFlags().StringVar(&flagStartDate, "start", "", "查询起始日期 (2006-01-02)")
	rootCmd.PersistentFlags().StringVar(&flagEndDate, "end", "", "查询结束日期 (2006-01-02)")

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(buyCmd)
	rootCmd.AddCommand(sellCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(cancelCmd)
	rootCmd.AddCommand(priceCmd)
	rootCmd.AddCommand(versionCmd)
}
