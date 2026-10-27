return function(a, b, f)
	local r = 0
	for i = a, b do
		r += f(i)
	end
	return r
end