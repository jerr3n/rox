-- A large stress file for the lexer and parser. It is valid Luau without
-- type annotations, string interpolation or attributes, and tries to touch
-- every token, statement and expression form at least once, plus the
-- awkward corners between them.

--[[ a long comment
	that spans
	several lines ]]
--[==[ a level-2 long comment with ]] and ]=] inside it ]==]
--[ not a long comment, just a short one that starts with --[
--[= also a short one
---- dashes all the way down ----

-----------------------------------------------------------------------------
-- Literals
-----------------------------------------------------------------------------

local n0, n1, n2, n3 = 0, 7, 10, 1234567890
local f0, f1, f2, f3 = 3.14, 0.5, .5, 10.0
local e0, e1, e2, e3 = 1e6, 1E6, 1e+6, 1e-6
local e4, e5, e6 = 2.5e10, 2.5E-3, .5e2
local h0, h1, h2, h3 = 0xff, 0XFF, 0x1F, 0xDEAD_BEEF
local b0, b1, b2 = 0b1010, 0B11, 0b1111_0000
local u0, u1 = 1_000_000, 3.141_592

local s0 = "double"
local s1 = 'single'
local s2 = ""
local s3 = ''
local s4 = "it's"
local s5 = 'say "hi"'
local s6 = "escaped \"quotes\" and 'single'"
local s7 = 'don\'t'
local s8 = "back\\slash, tab\t, newline\n, bell\a, nul\0"
local s9 = "\x41\x42\u{43}\u{1F600}\068\0659"
local s10 = "ends in a backslash \\"
local s11 = "line one\
line two"
local s12 = "skip \z     whitespace"
local s13 = [[long string]]
local s14 = [[]]
local s15 = [==[has ]] and ]=] inside]==]
local s16 = [[first
second
third]]
local s17 = [=[
leading newline is dropped by Luau]=]
local s18 = "-- not a comment"
local s19 = "--[[ also not a comment ]]"
local s20 = "[[ not a long string ]]"

local t, f, none = true, false, nil

-----------------------------------------------------------------------------
-- Operators and precedence
-----------------------------------------------------------------------------

local a, b, c, d, e = 1, 2, 3, 4, 5

local arith = a + b - c * d / e // 2 % 3
local pow = 2 ^ 3 ^ 2
local powParen = (2 ^ 3) ^ 2
local negPow = -a ^ 2
local negPowParen = (-a) ^ 2
local unaryChain = - - -a
local notChain = not not not t
local lenChain = #{ #"abc", ## "x" == nil }
local concat = "a" .. "b" .. "c" .. 1 .. 2
local concatParen = ("a" .. "b") .. "c"
local concatNum = 1 .. 2
local concatDot = a .. .5
local cmp = a < b and b <= c and c > d or d >= e and a == b or b ~= c
local logic = a or b and c or not d and e
local logicParen = (a or b) and (c or d)
local mixed = -a + b * -c ^ -d .. e
local cmpChain = (a < b) == (c < d)
local notEq = not a == b
local notEqParen = not (a == b)
local deep = ((((((((((a))))))))))
local longExpr = a + b + c + d + e + a * b * c * d * e - a / b / c / d / e
local noSpaces = a+b*c-d/e^2 ..e
local commentInside = a --[[ comment ]] + --[==[ another ]==] b
local commentBetween = a -- trailing comment
	+ b -- and another
	* c

-----------------------------------------------------------------------------
-- If-expressions
-----------------------------------------------------------------------------

local ie0 = if t then 1 else 2
local ie1 = if a > b then "gt" elseif a < b then "lt" else "eq"
local ie2 = if t then if f then 1 else 2 else 3
local ie3 = (if t then 1 else 2) + 10
local ie4 = if t then 1 else 2 + 10
local ie5 = { if t then "x" else "y", key = if f then 1 elseif t then 2 else 3 }
local ie6 = if if t then f else t then "weird" else "but legal"
local ie7 = if a then b elseif c then d elseif e then a elseif b then c else d

-----------------------------------------------------------------------------
-- Tables
-----------------------------------------------------------------------------

local tEmpty = {}
local tList = { 1, 2, 3 }
local tTrailComma = { 1, 2, 3, }
local tTrailSemi = { 1, 2, 3; }
local tSemis = { 1; 2; 3 }
local tNamed = { x = 1, y = 2, z = 3 }
local tKeyed = { [1] = "one", ["two"] = 2, [a + b] = 3, [f] = 4 }
local tMixed = { 1, x = 2, [3] = 3; 4, "five", [ [[six]] ] = 6 }
local tNested = { inner = { deep = { deeper = { deepest = {} } } } }
local tFuncs = {
	add = function(x, y) return x + y end,
	sub = function(x, y)
		return x - y
	end,
	[ "mul" ] = function(x, y) return x * y end,
	function() end,
}
local tVararg = { ... }
local tCalls = { f(), (f()), g(1)(2), h"s", h{}, h[[long]] }
local tKeywordish = { ["end"] = 1, ["local"] = 2, ["function"] = 3 }
local tLongKey = { [ [==[key]==] ] = 1 }
local tMatrix = {
	{ 1, 0, 0 },
	{ 0, 1, 0 },
	{ 0, 0, 1 },
}

-----------------------------------------------------------------------------
-- Suffixes: fields, indexes, calls, method calls
-----------------------------------------------------------------------------

print(a.b)
print(a.b.c.d.e)
print(a[1])
print(a[1][2][3])
print(a["key"].field[1]["other"])
print(a.b[c].d[e])
print(t[ [[key]] ])

f()
f(1)
f(1, 2, 3)
f(...)
f(a, ...)
f"string arg"
f'single arg'
f[[long arg]]
f[==[level arg]==]
f{ table, arg }
f{}
f "spaced"
f { spaced = true }
f()()()
f(1)(2)(3)
f"a""b""c"
f{}{}{}
f"a"{}[[b]](1)

obj:method()
obj:method(1, 2)
obj:method"str"
obj:method{ 1, 2 }
obj:method[[long]]
obj.field:method()
obj[1]:method()
obj:method():chain():again(1):done"x"
obj.a.b.c:d(e).f[g]:h{}

local m0 = ("abc"):upper()
local m1 = ({ 1, 2, 3 })[2]
local m2 = (function() return 1 end)()
local m3 = (f)(1)
local m4 = (obj):method()
local m5 = ("x"):rep(3):upper():lower()
local m6 = #("abc"):rep(2)
local m7 = -obj:value()
local m8 = not obj.flag

-----------------------------------------------------------------------------
-- Assignment
-----------------------------------------------------------------------------

x = 1
x, y = 1, 2
x, y, z = f()
x, y = y, x
a.b = 1
a.b.c = 2
a[1] = 3
a[1][2] = 4
a.b[c].d = 5
f().x = 6
f()[1] = 7
obj:method().field = 8
(a).b = 9
x, a.b, a[1], f().y = 1, 2, 3, 4
x, y, z = 1
x = 1, 2, 3

x += 1
x -= 1
x *= 2
x /= 2
x //= 2
x %= 3
x ^= 2
x ..= "suffix"
a.b += 1
a[1] -= 1
f().count += 1

local l0
local l1, l2, l3
local l4 = 1
local l5, l6 = 1
local l7, l8 = 1, 2, 3
local l9 = ...
local l10, l11 = f(), g()
local shadow = 1
local shadow = shadow + 1

-----------------------------------------------------------------------------
-- Semicolons and the ambiguous-call guard
-----------------------------------------------------------------------------

local semi = 1;
x = 1; y = 2; z = 3;
f(); g(); h();
local p = f; (g)()
local q = f; (g).x = 1
x = y; (z):m()
do end; do end
while f() do break; end
f()
;(g)()

-----------------------------------------------------------------------------
-- Functions
-----------------------------------------------------------------------------

function global() end
function global2(a, b, c) return a, b, c end
function varargs(...) return ... end
function mixedArgs(a, b, ...) return select("#", ...) end
function a.b() end
function a.b.c.d.e() end
function a:method() return self end
function a.b.c:method(x) return self, x end

local function localFn() end
local function localArgs(x, y) return x * y end
local function localVararg(...) local n = select("#", ...) return n end
local function recursive(n)
	if n <= 0 then
		return 0
	end
	return n + recursive(n - 1)
end

local anon = function() end
local anonArgs = function(a, b) return a + b end
local anonVararg = function(...) return { ... } end

local function outer()
	local function middle()
		local function inner()
			return function()
				return function()
					return "deep"
				end
			end
		end
		return inner
	end
	return middle
end

print(outer()()()()())

local function returnsNothing()
	return
end

local function returnsSemi()
	return;
end

local function returnsMany()
	return 1, "two", { 3 }, function() end, ...
end

local function returnsParen(...)
	return (...)
end

local function returnsCall()
	return f(g(h()))
end

local function earlyReturn(x)
	if x then
		return 1
	end
	do
		return 2
	end
end

-----------------------------------------------------------------------------
-- Control flow
-----------------------------------------------------------------------------

do
	local scoped = 1
	do
		local nested = scoped
		do
			local deepest = nested
		end
	end
end

while a < 10 do
	a += 1
end

while true do
	if a > 100 then
		break
	end
	a *= 2
end

while f() do end

repeat
	a -= 1
until a <= 0

repeat
	local stop = a > 5
until stop

repeat until true

if a then end

if a then
	f()
end

if a then
	f()
else
	g()
end

if a then
	f()
elseif b then
	g()
elseif c then
	h()
else
	i()
end

if a then if b then if c then f() end end end

if a and (b or c) and not d then
	f()
elseif (a or b) == c then
	g()
end

for i = 1, 10 do
	print(i)
end

for i = 10, 1, -1 do end
for i = 0, 1, 0.1 do end
for i = #t, 1, -1 do end
for i = f(), g(), h() do end
for i = a + b, c * d, -(e) do end

for k, v in pairs(t) do
	print(k, v)
end

for i, v in ipairs(t) do end
for k in next, t do end
for k, v in next, t, nil do end
for a, b, c, d, e in f() do end
for k, v in t do end

for i = 1, 10 do
	if i % 2 == 0 then
		continue
	end
	if i > 7 then
		break
	end
	print(i)
end

for _, v in t do
	while v do
		repeat
			if v then
				continue
			end
			v = nil
		until not v
		break
	end
end

while true do
	continue
end

repeat
	continue
until true

-- continue is only a keyword in statement position; elsewhere it's a name
local continue = 1
print(continue)
local t2 = { continue = continue }
t2.continue = 2
continue = 3
print(t2.continue, continue)

-----------------------------------------------------------------------------
-- Names that look like keywords, but aren't
-----------------------------------------------------------------------------

local self = {}
local type = "not a keyword"
local typeof = type
local export = 1
local _ = 2
local __ = 3
local _G2 = _G
local endless = 1
local ifx, thenx, elsex, localx, functionx = 1, 2, 3, 4, 5
local andy, orc, notable, nilly, trueish, falsey = 1, 2, 3, 4, 5, 6
local a1b2c3 = 123
local CamelCase, snake_case, SCREAMING = 1, 2, 3
print(endless, ifx, andy, orc, notable, nilly, trueish, falsey)

-----------------------------------------------------------------------------
-- A small program: a class, a queue, and an event system
-----------------------------------------------------------------------------

local Vector = {}
Vector.__index = Vector

function Vector.new(x, y)
	return setmetatable({ x = x or 0, y = y or 0 }, Vector)
end

function Vector.__add(lhs, rhs)
	return Vector.new(lhs.x + rhs.x, lhs.y + rhs.y)
end

function Vector.__eq(lhs, rhs)
	return lhs.x == rhs.x and lhs.y == rhs.y
end

function Vector.__tostring(v)
	return "(" .. v.x .. ", " .. v.y .. ")"
end

function Vector:length()
	return (self.x ^ 2 + self.y ^ 2) ^ 0.5
end

function Vector:normalized()
	local len = self:length()
	return if len == 0 then Vector.new() else Vector.new(self.x / len, self.y / len)
end

local Queue = {}
Queue.__index = Queue

function Queue.new()
	return setmetatable({ first = 1, last = 0, items = {} }, Queue)
end

function Queue:push(value)
	self.last += 1
	self.items[self.last] = value
end

function Queue:pop()
	if self.first > self.last then
		return nil
	end
	local value = self.items[self.first]
	self.items[self.first] = nil
	self.first += 1
	return value
end

function Queue:isEmpty()
	return self.first > self.last
end

local Signal = {}
Signal.__index = Signal

function Signal.new()
	return setmetatable({ handlers = {}, nextId = 0 }, Signal)
end

function Signal:connect(handler)
	self.nextId += 1
	local id = self.nextId
	self.handlers[id] = handler
	return {
		disconnect = function()
			self.handlers[id] = nil
		end,
	}
end

function Signal:fire(...)
	for _, handler in pairs(self.handlers) do
		task.spawn(handler, ...)
	end
end

local changed = Signal.new()
local conn = changed:connect(function(old, new)
	print(("changed from %s to %s"):format(tostring(old), tostring(new)))
end)
changed:fire(Vector.new(1, 2), Vector.new(3, 4) + Vector.new(1, 1))
conn.disconnect()

local queue = Queue.new()
for i = 1, 5 do
	queue:push(i * i)
end
while not queue:isEmpty() do
	local item = queue:pop()
	if item % 2 == 0 then
		continue
	end
	print(item)
end

-- breadth-first search over a grid
local function bfs(grid, start, goal)
	local rows, cols = #grid, #grid[1]
	local seen = { [start[1] * cols + start[2]] = true }
	local frontier = Queue.new()
	frontier:push({ start[1], start[2], 0 })
	local dirs = { { 1, 0 }, { -1, 0 }, { 0, 1 }, { 0, -1 } }
	while not frontier:isEmpty() do
		local node = frontier:pop()
		local r, c, dist = node[1], node[2], node[3]
		if r == goal[1] and c == goal[2] then
			return dist
		end
		for _, dir in ipairs(dirs) do
			local nr, nc = r + dir[1], c + dir[2]
			local key = nr * cols + nc
			if nr < 1 or nr > rows or nc < 1 or nc > cols then
				continue
			elseif grid[nr][nc] == 1 or seen[key] then
				continue
			end
			seen[key] = true
			frontier:push({ nr, nc, dist + 1 })
		end
	end
	return -1
end

print(bfs({
	{ 0, 0, 0, 1 },
	{ 1, 1, 0, 1 },
	{ 0, 0, 0, 0 },
}, { 1, 1 }, { 3, 1 }))

-- quicksort with an explicit partition and varargs plumbing
local function quicksort(arr, lo, hi, cmp)
	lo = lo or 1
	hi = hi or #arr
	cmp = cmp or function(x, y) return x < y end
	if lo >= hi then
		return arr
	end
	local pivot, i = arr[hi], lo - 1
	for j = lo, hi - 1 do
		if cmp(arr[j], pivot) then
			i += 1
			arr[i], arr[j] = arr[j], arr[i]
		end
	end
	arr[i + 1], arr[hi] = arr[hi], arr[i + 1]
	quicksort(arr, lo, i, cmp)
	quicksort(arr, i + 2, hi, cmp)
	return arr
end

local sorted = quicksort({ 5, 3, 8, 1, 9, 2, 7 })
print(table.concat(sorted, ", "))
print(table.concat(quicksort({ "pear", "apple", "fig" }, nil, nil, function(x, y)
	return #x < #y or (#x == #y and x < y)
end), " "))

-- a tiny state machine driven by a table of closures
local machine = {
	state = "idle",
	transitions = {
		idle = { start = "running" },
		running = { pause = "paused", stop = "idle" },
		paused = { resume = "running", stop = "idle" },
	},
}

function machine:send(event)
	local next = self.transitions[self.state][event]
	if next == nil then
		error(("bad event %q in state %q"):format(event, self.state), 2)
	end
	self.state = next
	return self
end

machine:send"start":send"pause":send"resume":send"stop"

local ok, err = pcall(function()
	machine:send("explode")
end)
print(ok, err)

-----------------------------------------------------------------------------
-- Whitespace and layout oddities
-----------------------------------------------------------------------------

local	tabs	=	1
local spaced   =   2
local
	split
	=
	3
f(
	1,
	2,
	3
)
local crlfish = 1 local onOneLine = 2 print(crlfish, onOneLine)
if a then f() elseif b then g() else h() end while false do end repeat until true
for i=1,2 do end for k,v in t do end
local function oneLiner(x) return x end local after = oneLiner(1)

-- the file ends with a return, and a comment after it with no newline
return Vector, Queue, Signal, machine -- bye