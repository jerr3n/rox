# TODO

## Lexer

Roughly in order of how much they matter.

- [ ] **Token positions.** `Pos` is commented out, so tokens and errors don't
  say which line or column they came from. Do this before starting the
  parser, since adding it later touches every token and error.
- [ ] **Interpolated strings**, e.g. `` `hello {name}` ``. The backtick is
  currently an unknown character. Needs new token kinds and some state,
  because `{ ... }` inside the string holds normal expressions.
- [ ] **Attributes**, e.g. `@native`, `@checked`. `@` is currently an
  unknown character.
- [ ] **Error handling.** Only the last error is kept and it has no
  location. Keep every error, each with its position.
- [ ] **Stricter numbers.** Malformed numbers like `0x`, `1e` and `123abc`
  are accepted or split into pieces; Luau reports them as malformed.
- [ ] **Non-ASCII identifiers.** Identifiers are checked byte by byte with
  `unicode.IsLetter`, so `é` becomes a partial identifier plus an unknown
  character. Luau rejects it outright. (Non-ASCII inside strings and
  comments is fine.)
- [ ] **Decode string contents.** A string token's text is the raw source,
  quotes and escapes included. Something (lexer or parser) needs to turn
  `"\x41"` into `A`.
- [ ] **Test coverage check.** Each `lexer/test/*.luau` file needs its own
  test function in `lexer_test.go`, so a new file isn't run until one is
  added. A test that fails when a file has no test would catch this.
