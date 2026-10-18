local Hello = require(script.Parent.Hello)

return function(name)
	Hello()
	print("hi " .. name)
end
