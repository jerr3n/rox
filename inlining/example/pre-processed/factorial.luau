return function(n)
	assert(n >= 0 and n % 1 == 0, "n must be a non-negative integer")
	local r = 1
	for i = 1, n do
		r = r * i
	end
	return r
end 