do
	local scoped = 1
end

while i < 10 do
	i += 1
end

repeat
	i -= 1
until i == 0

if a then
	x()
end

if a then
	x()
elseif b then
	y()
elseif c then
	z()
else
	w()
end

for i = 1, 10 do
	print(i)
end

for i = 10, 1, -1 do end

for k, v in pairs(t) do
	if v then
		continue
	end
	break
end

local pick = if a then 1 elseif b then 2 else 3
