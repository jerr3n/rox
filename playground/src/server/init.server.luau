local ReplicatedStorage = game:GetService("ReplicatedStorage")

local Hello = require(game:GetService("ReplicatedStorage").Shared.Hello)
local Greeter = require(ReplicatedStorage.Shared.Greeter) -- local alias: needs the scope resolver

print("Hello world, from server!")
Hello()
Greeter("server")
