package main

import "github.com/jerr3n/rox/modules/tables"

func main() {
	tables.TableInsertPatch(`
local mascots = {"cute gopher", "okay crab"}
table.insert(mascots, "ugly seal")
`)
	tables.TableInsertPatch(`
local array = {}
for i = 1, 10 do
	table.insert(array, i)
end
`)
}
