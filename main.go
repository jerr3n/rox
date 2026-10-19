package main

import (
	"log"
	"os"

	"github.com/jerr3n/rox/lexer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		log.Fatalf("zap failed to start, error %v", err)
	}
	data, err := os.ReadFile("main.luau")
	if err != nil {
		logger.Fatal("failed to read file", zap.Error(err))
	}
	logger.Info("read main.luau", zap.Int("length", len(data)))
	lexed, err := lexer.Lexer(string(data))
	if err != nil {
		logger.Fatal("failed to lex file", zap.Error(err))
	}
	logger.Info("lexed main.luau", zap.Int("toks", len(*lexed)))
}
