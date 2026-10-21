for numeratorCoefficient = 1, 2400 do
    local funcSinLn53Argument_1 = ((3.141592653589793 * numeratorCoefficient) / 1200)
    local funcSinLn53Variable_t = funcSinLn53Argument_1 * 0.15915494309189535
    local funcFloorLn45ReturnValue1 = nil
    local funcFloorLn45Argument1 = (funcSinLn53Variable_t + 0.5)
    if funcFloorLn45Argument1 == 1 / 0 or funcFloorLn45Argument1 == -1 / 0 then
        funcFloorLn45ReturnValue1 = funcFloorLn45Argument1
    else 
        funcFloorLn45ReturnValue1 = funcFloorLn45Argument1 - funcFloorLn45Argument1 % 1
    end
    local funcSinLn53Variable_z = (funcSinLn53Variable_t - funcFloorLn45ReturnValue1) * 6.283185307179586
    local funcSumLn47Variable_r = 0
    for funcSumLn47Variable_i = 0, 10 do

        local funcFactorialLn48ReturnValue1 = nil
        local funcFactorialLn48Argument1 = 2*funcSumLn47Variable_i+1
        assert(funcFactorialLn48Argument1 >= 0 and funcFactorialLn48Argument1 % 1 == 0, "n must be a non-negative integer")
        local funcFactorialLn48Variable_r = 1
        for funcFactorialLn48Variable_i = 1, funcFactorialLn48Argument1 do 
            funcFactorialLn48Variable_r = funcFactorialLn48Variable_r * funcFactorialLn48Variable_i
        end
        funcFactorialLn48ReturnValue1 = funcFactorialLn48Variable_r
        local funcPowLn48ReturnValue1_Call1 = -1 ^ funcSumLn47Variable_i
        local funcPowLn48ReturnValue1_Call2 = funcSinLn53Variable_z ^ 2*funcSumLn47Variable_i+1
        funcSumLn47Variable_r += funcPowLn48ReturnValue1_Call1 * funcPowLn48ReturnValue1_Call2 / funcFactorialLn48ReturnValue1
    end    
end