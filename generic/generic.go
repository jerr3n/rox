package generic

type Pos int
type LevelFault int

const (
	NoFault         LevelFault = -1
	LexerLevelFault LevelFault = iota
	ParserLevelFault
	ReadLevelFault
)
