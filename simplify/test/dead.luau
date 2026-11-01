-- Dead locals whose values still have side effects must keep them.
local count = 0
local function tick(tag)
	count += 1
	print("tick", tag, count)
	return count
end

local unusedCall = tick("unused local")
local a, b = tick("multi a"), tick("multi b")
local writeOnly
writeOnly = tick("write-only")

-- Dead helpers and closures with no effects.
local function helper(x)
	return x * 2
end
local closure = function()
	print("never runs")
end
local tbl = { 1, 2, k = 3 }

-- A shadowed write-only local must not touch the outer one.
local y = 5
do
	local y
	y = 3
end
print("y", y)

-- A partly dead assignment keeps the live half.
local p, q = 1, 2
p, q = 10, 20
print("p", p)

-- Values that may error stay put.
local ok = pcall(function()
	local boom = nil + 1
end)
print("pcall ok", ok)

print("count", count)
