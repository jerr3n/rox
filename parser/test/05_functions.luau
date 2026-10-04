function greet(name)
	print("hi " .. name)
end

function a.b.c(x, y) end

function obj:method(...)
	return self, ...
end

local function helper() end

local add = function(a, b)
	return a + b
end

local apply = function(f, ...) return f(...) end

print(function() end)
