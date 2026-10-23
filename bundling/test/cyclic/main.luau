-- main -> even -> odd -> even: can't be bundled, the bundler should report the cycle
local even = require("./even")

print(even(10))
