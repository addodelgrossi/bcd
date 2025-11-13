package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	flagYearMonth string
	flagWorkdir   string
	flagOutDB     string
	logger        *slog.Logger
	// Version is set via ldflags during build
	Version = "dev"
)

var RootCmd = &cobra.Command{
	Use:     "bcd",
	Short:   "Brazil Companies Database - CNPJ OpenData → SQLite loader",
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		lvl := new(slog.LevelVar)
		lvl.Set(slog.LevelInfo)
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
		logger.Info("boot", slog.String("cmd", cmd.Name()), slog.String("go", runtime.Version()))
		if flagYearMonth == "" {
			return fmt.Errorf("--ym (YYYY-MM) é obrigatório, ex: 2025-10")
		}
		if flagWorkdir == "" {
			flagWorkdir = "/tmp/cnpj_rf"
		}
		if flagOutDB == "" {
			flagOutDB = "./cnpj.sqlite"
		}
		if err := os.MkdirAll(flagWorkdir, 0o755); err != nil {
			return err
		}
		return nil
	},
}

func Execute() { _ = RootCmd.Execute() }

func init() {
	RootCmd.PersistentFlags().StringVar(&flagYearMonth, "ym", "", "ano-mês (YYYY-MM), ex: 2025-10")
	RootCmd.PersistentFlags().StringVar(&flagWorkdir, "workdir", "", "diretório de trabalho (default /tmp/cnpj_rf)")
	RootCmd.PersistentFlags().StringVar(&flagOutDB, "out", "", "arquivo SQLite de saída (default ./cnpj.sqlite)")
}
