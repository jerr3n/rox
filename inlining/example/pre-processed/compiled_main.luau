local InvTau = 0.15915494309189535
local Tau = 6.283185307179586
local Pi = 3.141592653589793
for numeratorCoefficient = 1, 2400 do
	do
		local n_5 = (Pi * numeratorCoefficient) / 1200
		local t_2 = n_5 * InvTau
		local floor_result
		do
			local n_6 = t_2 + 0.5
			if n_6 == 1 / 0 or n_6 == -1 / 0 then
				floor_result = n_6
			else
				floor_result = n_6 - n_6 % 1
			end
		end
		local z_2 = (t_2 - floor_result) * Tau
		local sum_result
		do
			local a_2 = 0
			local b_2 = 10
			local f_2 = function(j_2)
				local pow_result
				do
					local x_2 = -1
					local n_7 = j_2
					pow_result = x_2 ^ n_7
				end
				local pow_result_2
				do
					local x_3 = z_2
					local n_8 = 2 * j_2 + 1
					pow_result_2 = x_3 ^ n_8
				end
				local factorial_result
				do
					local n_9 = 2 * j_2 + 1
					assert(n_9 >= 0 and n_9 % 1 == 0, "n must be a non-negative integer")
					local r_4 = 1
					for i_4 = 1, n_9 do
						r_4 = r_4 * i_4
					end
					factorial_result = r_4
				end
				return pow_result * pow_result_2 / factorial_result
			end
			local r_3 = 0
			for i_3 = a_2, b_2 do
				local f_result
				do
					local j_3 = i_3
					local pow_result_3
					do
						local x_4 = -1
						local n_10 = j_3
						pow_result_3 = x_4 ^ n_10
					end
					local pow_result_4
					do
						local x_5 = z_2
						local n_11 = 2 * j_3 + 1
						pow_result_4 = x_5 ^ n_11
					end
					local factorial_result_2
					do
						local n_12 = 2 * j_3 + 1
						assert(n_12 >= 0 and n_12 % 1 == 0, "n must be a non-negative integer")
						local r_5 = 1
						for i_5 = 1, n_12 do
							r_5 = r_5 * i_5
						end
						factorial_result_2 = r_5
					end
					f_result = pow_result_3 * pow_result_4 / factorial_result_2
				end
				r_3 += f_result
			end
			sum_result = r_3
		end
	end
end
