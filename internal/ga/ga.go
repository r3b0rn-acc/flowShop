package ga

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"flowShop/internal/flowshop"
	"flowShop/internal/opt"
)

// Solver — реализация генетического алгоритма для задачи flow-shop.
type Solver struct {
	Cfg Config
	Rng *rand.Rand
}

// New возвращает новый GA-солвер с валидацией конфигурации, с использованием инициализированного генератора случайных чисел.
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

	// Проверка корректности входных данных и конфигурации
	if err := inst.Validate(); err != nil {
		return opt.Result{}, err
	}
	if err := s.Cfg.Validate(); err != nil {
		return opt.Result{}, err
	}
	if s.Rng == nil {
		return opt.Result{}, fmt.Errorf("генератор случайных чисел не инициализирован (nil)")
	}

	// Оценщик значения целевой функции для flow-shop задачи
	eval, err := flowshop.NewEvaluator(inst)
	if err != nil {
		return opt.Result{}, err
	}

	jobs := inst.Jobs
	popSize := s.Cfg.Population

	// Вспомогательная анонимная функция для создания двумерного массива перестановок
	makePerms := func() [][]int {
		backing := make([]int, popSize*jobs)
		perms := make([][]int, popSize)
		for i := 0; i < popSize; i++ {
			perms[i] = backing[i*jobs : (i+1)*jobs]
		}
		return perms
	}

	// Две популяции: текущая (A) и следующая (B)
	permsA := makePerms()
	permsB := makePerms()
	scoresA := make([]int, popSize)
	scoresB := make([]int, popSize)

	// Инициализация начальной популяции
	for i := 0; i < popSize; i++ {
		initPermutation(permsA[i])
		shufflePermutation(permsA[i], s.Rng)
		ms := eval.MustMakespan(permsA[i])
		scoresA[i] = ms
	}
	evaluations := popSize

	// Поиск лучшего решения в начальной популяции
	bestPerm := make([]int, jobs)
	bestMakespan := scoresA[0]
	copy(bestPerm, permsA[0])
	for i := 1; i < popSize; i++ {
		if scoresA[i] < bestMakespan {
			bestMakespan = scoresA[i]
			copy(bestPerm, permsA[i])
		}
	}

	// Массивы для кроссовера:
	// mark и stamp используются для отметки уже включённых работ
	mark := make([]int, jobs)
	stamp := 1

	// Временный буфер для второго потомка,
	// если в популяции остаётся нечётное число мест
	scratchChild := make([]int, jobs)

	// Индексы для сортировки популяции по приспособленности
	idxs := make([]int, popSize)
	for i := range idxs {
		idxs[i] = i
	}

	for gen := 0; gen < s.Cfg.Generations; gen++ {
		// Для поддержки отмены через context
		if err := ctx.Err(); err != nil {
			res := ToOptResult(
				bestPerm,
				bestMakespan,
				evaluations,
				gen,
				map[string]any{"stopped": "context"},
			)
			res.Duration = time.Since(start)
			return res, err
		}

		// Сортировка индексов по возрастанию значения целевой функции
		sort.Slice(idxs, func(i, j int) bool {
			return scoresA[idxs[i]] < scoresA[idxs[j]]
		})

		write := 0

		// Элитизм (переносим лучших особей без изменений)
		for e := 0; e < s.Cfg.Elite; e++ {
			src := idxs[e]
			copy(permsB[write], permsA[src])
			scoresB[write] = scoresA[src]
			write++
		}

		// Генерация остальных особей нового поколения
		for write < popSize {
			// Турнирный отбор
			p1 := tournamentSelect(scoresA, s.Cfg.TournamentSize, s.Rng)
			p2 := tournamentSelect(scoresA, s.Cfg.TournamentSize, s.Rng)
			if popSize > 1 {
				for p2 == p1 {
					p2 = tournamentSelect(scoresA, s.Cfg.TournamentSize, s.Rng)
				}
			}

			child1 := permsB[write]
			hasSecond := write+1 < popSize
			child2 := scratchChild
			if hasSecond {
				child2 = permsB[write+1]
			}

			// Кроссовер
			if s.Rng.Float64() < s.Cfg.CrossoverRate {
				orderCrossoverOX(
					permsA[p1],
					permsA[p2],
					child1,
					child2,
					s.Rng,
					mark,
					&stamp,
				)
			} else {
				copy(child1, permsA[p1])
				if hasSecond {
					copy(child2, permsA[p2])
				}
			}

			// Мутация
			if s.Rng.Float64() < s.Cfg.MutationRate {
				mutateSwap(child1, s.Rng)
			}
			if hasSecond && s.Rng.Float64() < s.Cfg.MutationRate {
				mutateSwap(child2, s.Rng)
			}

			// Оценка первого потомка
			ms1 := eval.MustMakespan(child1)
			scoresB[write] = ms1
			evaluations++
			if ms1 < bestMakespan {
				bestMakespan = ms1
				copy(bestPerm, child1)
			}
			write++

			// Оценка второго потомка
			if hasSecond {
				ms2 := eval.MustMakespan(child2)
				scoresB[write] = ms2
				evaluations++
				if ms2 < bestMakespan {
					bestMakespan = ms2
					copy(bestPerm, child2)
				}
				write++
			}
		}

		// Смена поколений
		permsA, permsB = permsB, permsA
		scoresA, scoresB = scoresB, scoresA
	}

	res := ToOptResult(
		bestPerm,
		bestMakespan,
		evaluations,
		s.Cfg.Generations,
		map[string]any{
			"population":  s.Cfg.Population,
			"generations": s.Cfg.Generations,
			"elite":       s.Cfg.Elite,
		},
	)
	res.Duration = time.Since(start)
	return res, nil
}
