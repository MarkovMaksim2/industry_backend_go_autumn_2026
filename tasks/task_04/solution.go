package main

import "math"

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	stats := Stats{0, 0, math.MaxInt64, math.MinInt64}

	if len(nums) <= 1 {
		stats.Max = 0
		stats.Min = 0
		return stats
	}

	stats.Count = len(nums) - 1

	for i := range len(nums) - 1 {
		stats.Min = min(stats.Min, nums[i+1]-nums[i])
		stats.Max = max(stats.Max, nums[i+1]-nums[i])
		stats.Sum += nums[i+1] - nums[i]
	}

	return stats
}
