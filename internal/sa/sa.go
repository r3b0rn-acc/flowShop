package sa

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"flowShop/internal/flowshop"
	"flowShop/internal/opt"
)

// Solver - структура реализации алгоритма имитации отжига
type Solver struct {
	Cfg Config
	Rng *rand.Rand
}

// New возвращает новый SA-солвер с валидацией конфигурации, с использованием инициализированного генератора случайных чисел.
// Используется в фабриках.
func New(cfg Config, rng *rand.Rand) (*Solver, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if rng == nil {
		return nil, fmt.Errorf("генератор случайных чисел не инициализирован (nil)")
	}
	return &Solver{Cfg: cfg, Rng: rng}, nil
}

// Solve — реализация эвристики.
func (s *Solver) Solve(ctx context.Context, inst *flowshop.Instance) (opt.Result, error) {
	start := time.Now()

	if err := inst.Validate(); err != nil {
		return opt.Result{}, err
	}
	if err := s.Cfg.Validate(); err != nil {
		return opt.Result{}, err
	}
	if s.Rng == nil {
		return opt.Result{}, fmt.Errorf("генератор случайных чисел не инициализирован (nil)")
	}

	// Оценка значения целевой функции для flow-shop задачи
	eval, err := flowshop.NewEvaluator(inst)
	if err != nil {
		return opt.Result{}, err
	}

	n := inst.Jobs

	maxIter := s.Cfg.Iterations
	if maxIter <= 0 {
		maxIter = s.Cfg.IterationsPerJob * n
	}

	// Текущее и кандидатное решения
	curr := make([]int, n)
	cand := make([]int, n)

	// Инициализация текущего решения
	initPermutation(curr)
	shufflePermutation(curr, s.Rng)

	currCost := eval.MustMakespan(curr)
	bestCost := currCost
	best := make([]int, n)
	copy(best, curr)

	evals := 1
	T := s.Cfg.InitialTemp

	for iter := 0; iter < maxIter && T > s.Cfg.FinalTemp; iter++ {
		// Для поддержки отмены через context
		if err := ctx.Err(); err != nil {
			return opt.Result{
				Permutation: best,
				Makespan:    bestCost,
				Evaluations: evals,
				Iterations:  iter,
				Duration:    time.Since(start),
				Meta: map[string]any{
					"stopped": "context",
					"T":       T,
				},
			}, err
		}

		copy(cand, curr)
		switch s.Cfg.Neighborhood {
		case NeighborhoodSwap:
			// Окрестность на основе обмена двух элементов
			neighborSwap(cand, s.Rng)
		case NeighborhoodInsert:
			// Окрестность на основе вставки элемента в другую позицию
			neighborInsert(cand, s.Rng)
		default:
			neighborSwap(cand, s.Rng)
		}

		candCost := eval.MustMakespan(cand)
		evals++

		delta := candCost - currCost
		accept := false
		if delta <= 0 {
			// Улучшающее решение принимаем всегда
			accept = true
		} else {
			// Критерий Метрополиса:
			// допускает принятие ухудшающих решений
			p := math.Exp(-float64(delta) / T)
			if s.Rng.Float64() < p {
				accept = true
			}
		}

		if accept {
			// Обмен ролей текущего и кандидатного решений
			curr, cand = cand, curr
			currCost = candCost

			// Обновление глобально лучшего решения
			if currCost < bestCost {
				bestCost = currCost
				copy(best, curr)
			}
		}

		// Охлаждение температуры
		T *= s.Cfg.Alpha
	}

	return opt.Result{
		Permutation: best,
		Makespan:    bestCost,
		Evaluations: evals,
		Iterations:  maxIter,
		Duration:    time.Since(start),
		Meta: map[string]any{
			"initial_temp": s.Cfg.InitialTemp,
			"final_temp":   s.Cfg.FinalTemp,
			"alpha":        s.Cfg.Alpha,
			"neighborhood": string(s.Cfg.Neighborhood),
		},
	}, nil
}

// initPermutation генерирует срез [0, 1, 2, ..., n-1].
// Используется как базовое состояние перед случайной перестановкой.
func initPermutation(p []int) {
	for i := range p {
		p[i] = i
	}
}

// shufflePermutation выполняет случайную перестановку элементов.
func shufflePermutation(p []int, rng *rand.Rand) {
	for i := len(p) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		p[i], p[j] = p[j], p[i]
	}
}

// Формирует соседнее решение путём обмена двух случайных позиций.
func neighborSwap(p []int, rng *rand.Rand) {
	if len(p) < 2 {
		return
	}
	i := rng.Intn(len(p))
	j := rng.Intn(len(p) - 1)
	if j >= i {
		j++
	}
	p[i], p[j] = p[j], p[i]
}

// Формирует соседнее решение путём извлечения элемента из позиции i и вставки его в позицию j.
func neighborInsert(p []int, rng *rand.Rand) {
	n := len(p)
	if n < 2 {
		return
	}
	i := rng.Intn(n)
	j := rng.Intn(n - 1)
	if j >= i {
		j++
	}

	// Перемещаем элемент из позиции i в позицию j
	val := p[i]
	if i < j {
		// Сдвиг элементов влево
		copy(p[i:j], p[i+1:j+1])
		p[j] = val
	} else {
		// Сдвиг элементов вправо
		copy(p[j+1:i+1], p[j:i])
		p[j] = val
	}
}
