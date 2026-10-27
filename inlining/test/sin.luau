local InvTau = 0.15915494309189535
local Tau = 6.283185307179586
local Pi = 3.141592653589793
local function floor(n)
    if n == 1 / 0 or n == -1 / 0 then
        return n
    end
    return n - n % 1
end
local function sum(a, b, f)
    local r = 0
    for i = a, b do
        r += f(i)
    end
    return r
end
local function pow(x, n)
    return x ^ n
end
local function factorial(n)
    assert(n >= 0 and n % 1 == 0, "n must be a non-negative integer")
    local r = 1
    for i = 1, n do
        r = r * i
    end
    return r
end
local function sin(n)
    local t = n * InvTau
    local z = (t - floor(t + 0.5)) * Tau
    return sum(0, 10, function(j)
        return pow(-1, j) * pow(z, 2*j+1) / factorial(2*j+1)
    end)
end
local acc = 0
for numeratorCoefficient = 1, 2400 do
    acc += sin((Pi * numeratorCoefficient)/1200)
end
print(string.format("%.10f", acc), sin(1), sin(-2))
