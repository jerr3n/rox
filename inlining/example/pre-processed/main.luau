local sin = require("./sin")
local consts = require("./constants")
for numeratorCoefficient = 1,  2400 do 
	sin((consts.Pi * numeratorCoefficient) / 1200 )
end
