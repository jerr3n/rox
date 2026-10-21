--[[
====================================================
Copyright (C) 1993 by Sun Microsystems, Inc. All rights reserved.

Developed at SunPro, a Sun Microsystems, Inc. business.
Permission to use, copy, modify, and distribute this
software is freely granted, provided that this notice
is preserved.
====================================================
]]
local consts = require("./constants")
local floor = require("./floor")
local sum = require("./sum")
local pow = require("./pow")
local factorial = require("./factorial")
return function(n) 
    local t = n * consts.InvTau
    local z = (t - floor(t + 0.5)) * consts.Tau

    sum(0, 10, function(j)
        return pow(-1, j) * pow(z, 2*j+1) / factorial(2*j+1)
    end)
end