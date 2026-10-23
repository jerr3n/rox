local even = require("./even")

return function(n)
	if n == 0 then
		return false
	end
	return even(n - 1)
end
