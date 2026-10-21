package main

import (
	"errors"
	"fmt"
	"log"
	"slices"

	"github.com/jerr3n/rox/modules/xfile"
	"github.com/jerr3n/rox/util"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type VisitStatus int

const (
	Unvisited VisitStatus = iota
	Visiting
	Visited
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
	sm := xfile.Sourcemap{}
	files := sm.Follow("main.luau")
	for _, file := range files {
		if file.Fault > util.NoFault {
			logger.Fatal("failed to parse all",
				zap.Int("fault level", int(file.Fault)),
				zap.Error(file.Err),
			)
		}
		logger.Debug("file okay", zap.String("file", file.Path))
		for _, require := range file.Requires {
			if require.Path == "" {
				logger.Fatal("blank require path",
					zap.String("path", file.Path),
					zap.String("require", fmt.Sprintf("%p", &require)),
				)
			}
			logger.Debug("require okay",
				zap.String("path", file.Path),
				zap.String("require", require.Path),
			)
		}
	}
	lookup := map[string]*xfile.File{}
	state := map[string]VisitStatus{}
	for _, file := range files {
		lookup[file.Path] = file
		state[file.Path] = Unvisited
	}
	var stack []string
	var cycle []string
	var output []*xfile.File
	var push func(path string)
	var pop func() string
	var visit func(path string) error
	push = func(path string) {
		stack = append(stack, path)
	}
	pop = func() string {
		last := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return last
	}
	visit = func(path string) error {
		status := state[path]
		switch status {
		case Visiting:
			i := slices.Index(stack, path)
			cycle = append(slices.Clone(stack[i:]), path)
			return errors.New("cyclic require")
		case Visited:
			return nil
		case Unvisited:
		}
		state[path] = Visiting
		push(path)
		file := lookup[path]
		for _, r := range file.Requires {
			visitErr := visit(r.Path)
			if visitErr != nil {
				return visitErr
			}
		}
		output = append(output, file)
		pop()
		state[path] = Visited
		return nil
	}
	err = visit(files[0].Path)
	if err != nil {
		logger.Fatal("error in visit", zap.Error(err), zap.Strings("cycle", cycle))
	}
	for _, t := range output {
		logger.Debug("output", zap.String("path", t.Path))
	}
	
	//spew.Dump(files)

}
