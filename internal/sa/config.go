package sa

import "fmt"

// Тип окрестности
type Neighborhood string

const (
	NeighborhoodSwap   Neighborhood = "swap"
	NeighborhoodInsert Neighborhood = "insert"
)

type Config struct {
	Iterations       int
	IterationsPerJob int

	InitialTemp float64
	FinalTemp   float64
	Alpha       float64

	Neighborhood Neighborhood
}

func DefaultConfig() Config {
	return Config{
		Iterations:       0,
		IterationsPerJob: 2500,

		InitialTemp: 2000.0,
		FinalTemp:   0.5,
		Alpha:       0.995,

		Neighborhood: NeighborhoodSwap,
	}
}

func (c Config) Validate() error {
	if c.Iterations <= 0 && c.IterationsPerJob <= 0 {
		return fmt.Errorf(
			"должно быть задано Iterations > 0 или IterationsPerJob > 0",
		)
	}
	if c.InitialTemp <= 0 {
		return fmt.Errorf(
			"InitialTemp должно быть > 0 (получено %f)",
			c.InitialTemp,
		)
	}
	if c.FinalTemp <= 0 {
		return fmt.Errorf(
			"FinalTemp должно быть > 0 (получено %f)",
			c.FinalTemp,
		)
	}
	if c.FinalTemp >= c.InitialTemp {
		return fmt.Errorf(
			"FinalTemp должно быть < InitialTemp (получено %f >= %f)",
			c.FinalTemp,
			c.InitialTemp,
		)
	}
	if c.Alpha <= 0 || c.Alpha >= 1 {
		return fmt.Errorf(
			"alpha должно лежать в интервале (0,1) (получено %f)",
			c.Alpha,
		)
	}
	switch c.Neighborhood {
	case NeighborhoodSwap, NeighborhoodInsert:
		// ok
	default:
		return fmt.Errorf(
			"неизвестный тип окрестности %q",
			c.Neighborhood,
		)
	}
	return nil
}
