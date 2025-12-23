package ga

import "fmt"

type Config struct {
	Population     int
	Generations    int
	Elite          int
	TournamentSize int
	CrossoverRate  float64
	MutationRate   float64
}

func (c Config) Validate() error {
	if c.Population <= 1 {
		return fmt.Errorf(
			"размер популяции должен быть > 1 (получено %d)",
			c.Population,
		)
	}
	if c.Generations <= 0 {
		return fmt.Errorf(
			"количество поколений должно быть > 0 (получено %d)",
			c.Generations,
		)
	}
	if c.Elite < 0 || c.Elite >= c.Population {
		return fmt.Errorf(
			"число элитных особей должно быть в диапазоне [0, population) (получено %d)",
			c.Elite,
		)
	}
	if c.TournamentSize <= 0 {
		return fmt.Errorf(
			"размер турнира должен быть > 0 (получено %d)",
			c.TournamentSize,
		)
	}
	if c.CrossoverRate < 0 || c.CrossoverRate > 1 {
		return fmt.Errorf(
			"вероятность кроссовера должна быть в диапазоне [0,1] (получено %f)",
			c.CrossoverRate,
		)
	}
	if c.MutationRate < 0 || c.MutationRate > 1 {
		return fmt.Errorf(
			"вероятность мутации должна быть в диапазоне [0,1] (получено %f)",
			c.MutationRate,
		)
	}
	return nil
}

func DefaultConfig() Config {
	return Config{
		Population:     150,
		Generations:    400,
		Elite:          4,
		TournamentSize: 5,
		CrossoverRate:  0.90,
		MutationRate:   0.15,
	}
}
