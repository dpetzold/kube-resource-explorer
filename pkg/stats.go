package main

import "sort"

func MinMax_int64(array []int64) (int64, int64) {
	var max int64 = array[0]
	var min int64 = array[0]
	for _, value := range array {
		if max < value {
			max = value
		}
		if min > value {
			min = value
		}
	}
	return min, max
}

func MinMax_float64(array []float64) (float64, float64) {
	var max float64 = array[0]
	var min float64 = array[0]
	for _, value := range array {
		if max < value {
			max = value
		}
		if min > value {
			min = value
		}
	}
	return min, max
}

func average_float64(array []float64) float64 {
	var total float64
	for _, value := range array {
		total += value
	}
	return total / float64(len(array))
}

func average_int64(array []int64) float64 {
	var total int64
	for _, value := range array {
		total += value
	}
	return float64(total) / float64(len(array))
}

func mode_int64(set map[int64]int) int64 {

	type KV struct {
		Key   int64
		Value int
	}

	ss := make([]KV, len(set))
	for k, v := range set {
		ss = append(ss, KV{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	return ss[0].Key
}
