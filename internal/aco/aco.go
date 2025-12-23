package aco

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"flowShop/internal/flowshop"
	"flowShop/internal/opt"
)

// Solver - структура реализации муравьиного алгоритма.
type Solver struct {
	Cfg Config
	Rng *rand.Rand
}

// New возвращает новый ACO-солвер с валидацией конфигурации, с использованием инициализированного генератора случайных чисел.
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
	startTime := time.Now()

	// Валидация входных данных
	if err := inst.Validate(); err != nil {
		return opt.Result{}, err
	}
	if err := s.Cfg.Validate(); err != nil {
		return opt.Result{}, err
	}
	if s.Rng == nil {
		return opt.Result{}, fmt.Errorf("генератор случайных чисел не инициализирован (nil)")
	}

	// Оценка целевой функции
	eval, err := flowshop.NewEvaluator(inst)
	if err != nil {
		return opt.Result{}, err
	}

	n := inst.Jobs

	maxIter := s.Cfg.Iterations
	if maxIter <= 0 {
		maxIter = s.Cfg.IterationsPerJob * n
	}

	ants := s.Cfg.Ants
	if ants < 1 {
		ants = 1
	}

	// чем быстрее работа — тем лучше
	eta := make([]float64, n)
	for j := 0; j < n; j++ {
		sum := 0
		for m := 0; m < inst.Machines; m++ {
			sum += inst.Time(j, m)
		}
		eta[j] = 1.0 / float64(sum+1)
	}

	// Матрица феромонов
	tau := make([]float64, (n+1)*n)
	for i := range tau {
		tau[i] = s.Cfg.Tau0
	}

	// Вспомогательные буферы
	perm := make([]int, n)        // текущая перестановка
	available := make([]int, n)   // доступные работы
	weights := make([]float64, n) // веса вероятностного выбора

	bestPerm := make([]int, n)
	bestCost := math.MaxInt
	evals := 0

	alpha := s.Cfg.Alpha
	beta := s.Cfg.Beta
	rho := s.Cfg.Rho
	Q := s.Cfg.Q

	for iter := 0; iter < maxIter; iter++ {
		// Для поддержки отмены через context
		if err := ctx.Err(); err != nil {
			return opt.Result{
				Permutation: bestPerm,
				Makespan:    bestCost,
				Evaluations: evals,
				Iterations:  iter,
				Duration:    time.Since(startTime),
				Meta: map[string]any{
					"stopped": "context",
				},
			}, err
		}

		// Лучшее решение текущей итерации
		iterBestCost := math.MaxInt
		iterBestPerm := make([]int, n)

		// Муравьи пошли
		for a := 0; a < ants; a++ {
			constructPermutation(
				n, tau, eta,
				alpha, beta,
				s.Cfg.CandidateK,
				s.Rng,
				perm, available, weights,
			)

			cost := eval.MustMakespan(perm)
			evals++

			// Локальное лучшее за итерацию
			if cost < iterBestCost {
				iterBestCost = cost
				copy(iterBestPerm, perm)
			}
			// Глобальное лучшее за всё время
			if cost < bestCost {
				bestCost = cost
				copy(bestPerm, perm)
			}
		}

		// Испарение феромона
		ev := 1.0 - rho
		for i := range tau {
			tau[i] *= ev
			if tau[i] < 1e-12 {
				tau[i] = 1e-12
			}
		}

		// Добавление феромона только по лучшему пути итерации
		dep := Q / float64(iterBestCost)
		addPheromonePath(tau, n, iterBestPerm, dep)
	}

	return opt.Result{
		Permutation: bestPerm,
		Makespan:    bestCost,
		Evaluations: evals,
		Iterations:  maxIter,
		Duration:    time.Since(startTime),
		Meta: map[string]any{
			"ants":        ants,
			"alpha":       alpha,
			"beta":        beta,
			"rho":         rho,
			"Q":           Q,
			"tau0":        s.Cfg.Tau0,
			"candidate_k": s.Cfg.CandidateK,
		},
	}, nil
}

func tauIdx(n, from, to int) int {
	return from*n + to
}

// addPheromonePath усиливает феромон вдоль полного пути перестановки.
// от фиктивного старта до последней работы.
func addPheromonePath(tau []float64, n int, perm []int, delta float64) {
	if len(perm) == 0 {
		return
	}
	start := n
	first := perm[0]
	tau[tauIdx(n, start, first)] += delta
	for i := 0; i < len(perm)-1; i++ {
		from := perm[i]
		to := perm[i+1]
		tau[tauIdx(n, from, to)] += delta
	}
}

// constructPermutation строит одну перестановку работ.
// На каждом шаге выбирается следующая работа вероятностно по формуле ACO.
func constructPermutation(
	n int,
	tau []float64,
	eta []float64,
	alpha float64,
	beta float64,
	candidateK int,
	rng *rand.Rand,
	outPerm []int,
	available []int,
	weights []float64,
) {
	for i := 0; i < n; i++ {
		available[i] = i
	}
	rem := n

	prev := n // prev — предыдущая вершина

	for pos := 0; pos < n; pos++ {
		// Ограничение списка кандидатов
		k := rem
		if candidateK > 0 && candidateK < rem {
			k = candidateK
			for t := 0; t < k; t++ {
				r := t + rng.Intn(rem-t)
				available[t], available[r] = available[r], available[t]
			}
		}

		// Подсчёт весов вероятностей выбора
		sumW := 0.0
		for i := 0; i < k; i++ {
			j := available[i]
			t := tau[tauIdx(n, prev, j)]

			// Формула ACO
			w := fastPow(t, alpha) * fastPow(eta[j], beta)
			weights[i] = w
			sumW += w
		}

		// Стохастический выбор следующей работы
		var chosenIdx int
		if sumW <= 0 {
			chosenIdx = rng.Intn(k)
		} else {
			r := rng.Float64() * sumW
			acc := 0.0
			chosenIdx = k - 1
			for i := 0; i < k; i++ {
				acc += weights[i]
				if r <= acc {
					chosenIdx = i
					break
				}
			}
		}

		job := available[chosenIdx]
		outPerm[pos] = job
		prev = job

		// Удаляем выбранную работу из списка доступных
		available[chosenIdx], available[rem-1] =
			available[rem-1], available[chosenIdx]
		rem--
	}
}

// fastPow — оптимизация для частых степеней.
// Таким образом избегаем вызова math.Pow в простых случаях.
func fastPow(x, p float64) float64 {
	if p == 0 {
		return 1.0
	}
	if p == 1 {
		return x
	}
	if p == 2 {
		return x * x
	}
	return math.Pow(x, p)
}
