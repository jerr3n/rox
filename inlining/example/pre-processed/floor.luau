--[[
====================================================
Copyright (C) 1993 by Sun Microsystems, Inc. All rights reserved.

Developed at SunPro, a Sun Microsystems, Inc. business.
Permission to use, copy, modify, and distribute this
software is freely granted, provided that this notice
is preserved.
====================================================
]]
return function(n)
	if n == 1 / 0 or n == -1 / 0 then
		return n
	end
	return n - n % 1
end