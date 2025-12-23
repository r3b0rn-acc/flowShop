package bench

import "math"

type IntStats struct {
	N    int
	Best int
	Mean float64
	Std  float64
}

func CalcIntStats(values []int) IntStats {
	s := IntStats{N: len(values)}
	if s.N == 0 {
		return s
	}

	best := values[0]
	sum := 0.0
	for _, v := range values {
		if v < best {
			best = v
		}
		sum += float64(v)
	}
	mean := sum / float64(s.N)

	variance := 0.0
	if s.N >= 2 {
		for _, v := range values {
			d := float64(v) - mean
			variance += d * d
		}
		variance /= float64(s.N - 1)
	}

	s.Best = best
	s.Mean = mean
	s.Std = math.Sqrt(variance)
	return s
}

type FloatStats struct {
	N    int
	Best float64
	Mean float64
	Std  float64
}

func CalcFloatStats(values []float64) FloatStats {
	s := FloatStats{N: len(values)}
	if s.N == 0 {
		return s
	}

	best := values[0]
	sum := 0.0
	for _, v := range values {
		if v < best {
			best = v
		}
		sum += v
	}
	mean := sum / float64(s.N)

	variance := 0.0
	if s.N >= 2 {
		for _, v := range values {
			d := v - mean
			variance += d * d
		}
		variance /= float64(s.N - 1)
	}

	s.Best = best
	s.Mean = mean
	s.Std = math.Sqrt(variance)
	return s
}
