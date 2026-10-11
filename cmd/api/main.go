// Package main es el entrypoint del backend de ByteBalance.
// Solo contiene el wiring de dependencias; la lógica de negocio vive en internal/.
package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("bytebalance api iniciada")
}
