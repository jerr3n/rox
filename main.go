package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jerr3n/rox/bundling"
	"github.com/jerr3n/rox/modules/xfile"
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
	defer logger.Sync()
	//data, err := os.ReadFile("main.luau")
	//if err != nil {
	//	logger.Fatal("failed to read file", zap.Error(err))
	//}
	//logger.Info("read main.luau", zap.Int("length", len(data)))
	//lexed, err := lexer.Lexer(string(data))
	//if err != nil {
	//	logger.Fatal("failed to lex file", zap.Error(err))
	//}
	//logger.Info("lexed main.luau", zap.Int("toks", len(*lexed)))
	//_, err = parser.Parser(*lexed)
	//if err != nil {
	//	logger.Fatal("failed to parse file", zap.Error(err))
	//}
	//logger.Info("parsed main.luau")
	//info := scope.Resolve(parsed)
	entry := "main.luau"
	if len(os.Args) > 1 {
		entry = os.Args[1]
	}
	out, err := bundling.Bundle(&xfile.Sourcemap{}, entry, logger)
	if err != nil {
		logger.Fatal("bundle failed", zap.Error(err))
	}
	// zap logs to stderr, so stdout is only the bundle
	fmt.Print(out)
}
