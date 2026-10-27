package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jerr3n/rox/ast"
	"github.com/jerr3n/rox/bundling"
	"github.com/jerr3n/rox/hoisting"
	"github.com/jerr3n/rox/inlining"
	"github.com/jerr3n/rox/xfile"
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
	wrap := flag.Bool("wrap", false, "keep every module in its own function instead of hoisting them into one scope")
	inline := flag.Bool("inline", true, "inline calls to local functions after hoisting; -inline=false stops after hoisting")
	output := flag.String("o", "", `file to write the output to, or "-" for stdout (default compiled_<entry> next to the entry, hoist_<entry> with -inline=false, or bundle_<entry> with -wrap)`)
	flag.Parse()
	entry := "main.luau"
	if flag.NArg() > 0 {
		entry = flag.Arg(0)
	}
	var passes []func(*ast.Block) *ast.Block
	prefix := "bundle_"
	if !*wrap {
		passes = append(passes, hoisting.Unwrap, hoisting.InlineAliases, hoisting.SplitTables)
		prefix = "hoist_"
		if *inline {
			passes = append(passes, inlining.Inline)
			prefix = "compiled_"
		}
	}
	out, err := bundling.Bundle(&xfile.Sourcemap{}, entry, logger, passes...)
	if err != nil {
		logger.Fatal("bundle failed", zap.Error(err))
	}
	if *output == "-" {
		// zap logs to stderr, so stdout is only the bundle
		fmt.Print(out)
		return
	}
	path := *output
	if path == "" {
		path = filepath.Join(filepath.Dir(entry), prefix+filepath.Base(entry))
	}
	// the sources are only read, never changed; only path is written
	if filepath.Clean(path) == filepath.Clean(entry) {
		logger.Fatal("output would overwrite the entry", zap.String("path", path))
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		logger.Fatal("write failed", zap.String("path", path), zap.Error(err))
	}
	logger.Info("wrote output", zap.String("path", path), zap.Int("bytes", len(out)))
}
