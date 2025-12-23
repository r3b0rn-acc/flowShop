package ts

import "fmt"

// Neighborhood определяет тип окрестности.
type Neighborhood string

const (
	NeighborhoodInsert Neighborhood = "insert"
	NeighborhoodSwap   Neighborhood = "swap"
)

type Config struct {
	Iterations       int
	IterationsPerJob int

	TabuTenure int

	TabuTenureRand int

	NeighborsPerIter int

	Neighborhood Neighborhood
}

func DefaultConfig() Config {
	return Config{
		Iterations:       0,
		IterationsPerJob: 250,

		TabuTenure:     7,
		TabuTenureRand: 3,

		NeighborsPerIter: 90,
		Neighborhood:     NeighborhoodInsert,
	}
}

func (c Config) Validate() error {
	if c.Iterations <= 0 && c.IterationsPerJob <= 0 {
		return fmt.Errorf(
			"должно быть задано Iterations > 0 или IterationsPerJob > 0",
		)
	}
	if c.TabuTenure <= 0 {
		return fmt.Errorf(
			"TabuTenure должно быть > 0 (получено %d)",
			c.TabuTenure,
		)
	}
	if c.TabuTenureRand < 0 {
		return fmt.Errorf(
			"TabuTenureRand должно быть >= 0 (получено %d)",
			c.TabuTenureRand,
		)
	}
	if c.NeighborsPerIter <= 0 {
		return fmt.Errorf(
			"NeighborsPerIter должно быть > 0 (получено %d)",
			c.NeighborsPerIter,
		)
	}
	switch c.Neighborhood {
	case NeighborhoodInsert, NeighborhoodSwap:
		// ok
	default:
		return fmt.Errorf(
			"неизвестный тип окрестности %q",
			c.Neighborhood,
		)
	}
	return nil
}
