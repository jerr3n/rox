# Contributing

---

In Rox, we have a handful of ways to write code. Rox's structure is simple, and follows this guide:

- Core parts of the program go where they should
  - For example: the primary AST function is in `ast/ast.go`.
- How something is used determines where it will go
  - If something is only used in one place, then it will reside where it is used.
  - If something is used more in one place then in another (or there are 2 different systems that seem to do the same thing), they will reside with the one that uses it more.
- Things that are used in different parts of the program are housed in `generic`.
  - E.g. `Pos` is used in the Lexer, AST, and Parser. It is housed in `generic`.