--!strict
-- A small inventory module that touches a bit of everything.

export type Item = {
	name: string,
	count: number,
	tags: {string}?,
}

type Id = string | number
type Both = Item & { id: Id }

local Inventory = {}
Inventory.__index = Inventory

function Inventory.new(capacity: number?)
	local self = setmetatable({}, Inventory)
	self.items = {} :: {Item}
	self.capacity = capacity or 10
	return self
end

function Inventory:add(name: string, count: number): boolean
	if #self.items >= self.capacity then
		return false
	elseif count <= 0 then
		error("count must be positive: " .. tostring(count))
	end
	for _, item in self.items do
		if item.name == name then
			item.count += count
			return true
		end
	end
	table.insert(self.items, { name = name, count = count })
	return true
end

local function first<T>(list: {T}): T?
	return list[1]
end

local function total(...: Item): number
	local sum = 0
	for i = 1, select("#", ...) do
		local item = select(i, ...)
		sum += item.count
	end
	return sum
end

local callback: (string, number) -> () = function(msg, code)
	local ok = code ~= 0 and not (code > 255)
	repeat
		code //= 2
	until code < 1
	while true do
		break
	end
	print(msg, ok, 2 ^ 8 % 3, -code, nil, first, total)
end

callback('done', 0x_FF)

return Inventory
