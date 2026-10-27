local log = {}
local function note(x) table.insert(log, tostring(x)) return x end

local function early(n)
	if n < 0 then
		return "neg"
	elseif n == 0 then
		note("zero")
	end
	local m = n * 2
	if m > 10 then
		return "big"
	end
	note("tail")
	return m
end

local function two(a, b) return a + 1, b + 1 end
local function none(x) note("none" .. x) end
local function bare(x) if x then return end note("bare") end
local function side(x) note("side" .. x) return x end
local function shadowed() return note end
local function rec(n) if n <= 0 then return 0 end return n + rec(n - 1) end
local function va(...) return select("#", ...) end
local function apply(f, x) return f(x) end

for _, v in ipairs({-1, 0, 3, 7}) do print("early", v, early(v)) end
print("two", two(1, 2))
local p, q = two(10, 20)
print("pq", p, q)
local t = {two(1, 1)}
print("table", #t)
print("multi-arg", select("#", two(1, 1)))
none("a")
print("none-val", none("b"))
print("count", select("#", none("c")))
bare(true) bare(false)
print("order", side("1") .. side("2"))
print("andor", false and side("x"), true or side("y"), nil or side("z"))
local i = 0
while side("w") and i < 2 do i += 1 end
if side("c1") == "no" then print("no") elseif side("c2") == "c2" then print("yes") end
do
	local note = function() return "inner" end
	print("capture", shadowed() == note, note())
end
print("rec", rec(4))
print("va", va(1, nil, 3))
print("apply", apply(function(x) return x * 3 end, 5))
local obj = {k = 2}
function obj:get(x) return self.k + side(x) end
print("method", obj:get(1))
print("ifexpr", if side("q") then side("r") else side("s"))
print("args-extra", side(1, side(2)))
print(table.concat(log, ","))
