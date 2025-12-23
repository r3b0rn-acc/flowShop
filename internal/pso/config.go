package pso

import "fmt"

type Config struct {
	Iterations       int
	IterationsPerJob int

	Particles int

	W  float64
	C1 float64
	C2 float64

	VMax float64

	PosMin float64
	PosMax float64
}

func DefaultConfig() Config {
	return Config{
		Iterations:       0,
		IterationsPerJob: 180,

		Particles: 60,

		W:  0.729,
		C1: 1.49445,
		C2: 1.49445,

		VMax:   0.25,
		PosMin: 0.0,
		PosMax: 1.0,
	}
}

func (c Config) Validate() error {
	if c.Iterations <= 0 && c.IterationsPerJob <= 0 {
		return fmt.Errorf(
			"должно быть задано Iterations > 0 или IterationsPerJob > 0",
		)
	}
	if c.Particles <= 0 {
		return fmt.Errorf(
			"Particles должно быть > 0 (получено %d)",
			c.Particles,
		)
	}
	if c.W < 0 {
		return fmt.Errorf(
			"W должно быть >= 0 (получено %f)",
			c.W,
		)
	}
	if c.C1 < 0 || c.C2 < 0 {
		return fmt.Errorf(
			"C1 и C2 должны быть >= 0 (получено %f, %f)",
			c.C1,
			c.C2,
		)
	}
	if c.PosMin >= c.PosMax {
		if !(c.PosMin == 0 && c.PosMax == 0) {
			return fmt.Errorf(
				"для ограничения PosMin должно быть < PosMax (получено %f >= %f)",
				c.PosMin,
				c.PosMax,
			)
		}
	}
	return nil
}
