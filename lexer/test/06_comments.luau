-- a short comment
local a = 1 -- trailing comment
--no space after the dashes
---three dashes
-- code in a comment is ignored: local b = 2 "string" [[x]]
--[[ a long
comment ]]
local c = 3
--[==[ long comment with ]] and ]=] inside ]==]
--[[]]local d = 4
--[ only one bracket, so this is a short comment ]]
--[= also a short comment
e = 5 --[[ inline ]] + 6
-- the last line has no newline