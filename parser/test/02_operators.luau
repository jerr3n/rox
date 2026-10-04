-- precedence: * before +
local a = 1 + 2 * 3
local b = 1 * 2 + 3
-- left-associative
local c = a - b - c
-- right-associative: ^ and ..
local d = 2 ^ 3 ^ 2
local e = "a" .. "b" .. "c"
-- unary binds tighter than everything except ^
local f = -x ^ 2
local g = -x + 1
local h = not a == b
local i = #t + 1
local j = 2 ^ -3
-- parentheses override precedence
local k = (1 + 2) * 3
-- and binds tighter than or, comparisons tighter than both
local l = a or b and c
local m = x < 10 and y >= 0 or z ~= nil
local n = 7 // 2 % 3
