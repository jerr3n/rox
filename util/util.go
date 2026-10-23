package util

type Pos int
type LevelFault int

const (
	NoFault         LevelFault = -1
	LexerLevelFault LevelFault = iota
	ParserLevelFault
	ReadLevelFault
)

func IsWordChar(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
