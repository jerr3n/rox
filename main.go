package main

import (
	"errors"
	"fmt"
	"log"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

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
	names := map[string]string{}
	for _, t := range output {
		path := t.Path
		logger.Debug("output", zap.String("path", path))
		ext := filepath.Ext(path)
		res := strings.TrimSuffix(path, ext)
		var sanitized strings.Builder
		for _, c := range res {
			if c <= utf8.RuneSelf && util.IsWordChar(byte(c)) {
				sanitized.WriteString(string(c))
			} else {
				sanitized.WriteString("_") // do not write your codebase primarily in anything that does not use a latin character set/english character set
			}
		}
		prefixed := fmt.Sprintf("__mod_%s", sanitized.String())
		vals := slices.Collect(maps.Values(names))
		if slices.Contains(vals, prefixed) {
			count := 1
			for _, v := range vals {
				if strings.HasPrefix(v, prefixed) { // this will not shoot you in the foot unless you make it
					count++
				} // if theres 2 functions in your code called something like 3dmakearray and 3dappendarray i dont feel bad for you
			}
			names[path] = fmt.Sprintf("%s_instance%d", prefixed, count)
		} else {
			names[path] = prefixed
		}
	}

	//spew.Dump(files)

}
