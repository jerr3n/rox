-- A functional-style program with no type annotations: higher-order
-- functions, closures, currying, recursion and varargs.

local function map(list, f)
	local out = {}
	for i, v in ipairs(list) do
		out[i] = f(v, i)
	end
	return out
end

local function filter(list, pred)
	local out = {}
	for _, v in ipairs(list) do
		if pred(v) then
			table.insert(out, v)
		end
	end
	return out
end

local function reduce(list, f, acc)
	for _, v in ipairs(list) do
		acc = f(acc, v)
	end
	return acc
end

local function compose(...)
	local fns = { ... }
	return function(x)
		for i = #fns, 1, -1 do
			x = fns[i](x)
		end
		return x
	end
end

local function curry(f)
	return function(a)
		return function(b)
			return f(a, b)
		end
	end
end

local function range(lo, hi, step)
	step = step or 1
	local out = {}
	for i = lo, hi, step do
		out[#out + 1] = i
	end
	return out
end

-- recursion, with an if-expression for the base case
local function fib(n)
	return if n < 2 then n else fib(n - 1) + fib(n - 2)
end

local function memoize(f)
	local cache = setmetatable({}, { __mode = "k" })
	return function(n)
		local hit = cache[n]
		if hit == nil then
			hit = f(n)
			cache[n] = hit
		end
		return hit
	end
end

-- a counter built from a closure over a local
local function counter(start)
	local n = start or 0
	return {
		next = function()
			n += 1
			return n
		end,
		reset = function()
			n = start or 0
		end,
	}
end

local add = curry(function(a, b) return a + b end)
local mul = curry(function(a, b) return a * b end)
local isEven = function(x) return x % 2 == 0 end
local square = function(x) return x ^ 2 end

local addThenDouble = compose(mul(2), add(10))
local evens = filter(range(1, 20), isEven)
local squares = map(evens, square)
local total = reduce(squares, function(a, b) return a + b end, 0)

local labels = map(range(0, 9), function(n, i)
	return tostring(i) .. ": " .. tostring(memoize(fib)(n))
end)

local c = counter(100)
c.next()
c.next()

print(addThenDouble(5), total, #labels, c.next())
print(table.concat(labels, "\n"))
print(reduce({ "a", "b", "c" }, function(s, x) return s .. x end, ""):upper())
