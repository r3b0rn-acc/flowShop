package aco

import "fmt"

type Config struct {
	Iterations       int
	IterationsPerJob int

	Ants int

	Alpha float64
	Beta  float64

	Rho float64

	Q float64

	Tau0 float64

	CandidateK int
}

func DefaultConfig() Config {
	return Config{
		Iterations:       0,
		IterationsPerJob: 120,

		Ants: 35,

		Alpha: 1.0,
		Beta:  2.0,

		Rho: 0.20,
		Q:   1000.0,

		Tau0: 1.0,

		CandidateK: 0,
	}
}

func (c Config) Validate() error {
	if c.Iterations <= 0 && c.IterationsPerJob <= 0 {
		return fmt.Errorf(
			"должно быть задано Iterations > 0 или IterationsPerJob > 0",
		)
	}
	if c.Ants <= 0 {
		return fmt.Errorf(
			"ants должно быть > 0 (получено %d)",
			c.Ants,
		)
	}
	if c.Alpha < 0 {
		return fmt.Errorf(
			"alpha должно быть >= 0 (получено %f)",
			c.Alpha,
		)
	}
	if c.Beta < 0 {
		return fmt.Errorf(
			"beta должно быть >= 0 (получено %f)",
			c.Beta,
		)
	}
	if c.Rho <= 0 || c.Rho >= 1 {
		return fmt.Errorf(
			"rho должно лежать в интервале (0,1) (получено %f)",
			c.Rho,
		)
	}
	if c.Q <= 0 {
		return fmt.Errorf(
			"Q должно быть > 0 (получено %f)",
			c.Q,
		)
	}
	if c.Tau0 <= 0 {
		return fmt.Errorf(
			"tau0 должно быть > 0 (получено %f)",
			c.Tau0,
		)
	}
	if c.CandidateK < 0 {
		return fmt.Errorf(
			"CandidateK должно быть >= 0 (получено %d)",
			c.CandidateK,
		)
	}
	return nil
}
