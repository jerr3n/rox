local odd = require("./odd")

return function(n)
	if n == 0 then
		return true
	end
	return odd(n - 1)
end
