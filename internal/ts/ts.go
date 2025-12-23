package ts

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"flowShop/internal/flowshop"
	"flowShop/internal/opt"
)

// maxInt используется как бесконечность для стоимостей.
const maxInt = int(^uint(0) >> 1)

// Solver - структура реализации муравьиного алгоритма.
type Solver struct {
	Cfg Config
	Rng *rand.Rand
}

// New возвращает новый TS-солвер с валидацией конфигурации, с использованием инициализированного генератора случайных чисел.
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

// Solve — основной цикл алгоритма
func (s *Solver) Solve(ctx context.Context, inst *flowshop.Instance) (opt.Result, error) {
	start := time.Now()

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

	// Текущее и кандидатное решения
	curr := make([]int, n)
	cand := make([]int, n)

	// Инициализация начального решения
	initPermutation(curr)
	shufflePermutation(curr, s.Rng)

	currCost := eval.MustMakespan(curr)
	evals := 1

	// Глобально лучшее решение
	best := make([]int, n)
	copy(best, curr)
	bestCost := currCost

	// Табу-список - кольцевой буфер с мапой
	// Ёмкость выбирается с запасом относительно длины табу
	tabu := newTabuList(max(32, (s.Cfg.TabuTenure+s.Cfg.TabuTenureRand)*4))

	neighbors := s.Cfg.NeighborsPerIter
	if neighbors < 1 {
		neighbors = 1
	}

	for iter := 0; iter < maxIter; iter++ {
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
				},
			}, err
		}

		// Лучший допустимый ход
		bestMoveFrom, bestMoveTo := -1, -1
		bestMoveCost := maxInt
		bestMoveKey := uint64(0)
		bestMoveJob := -1

		// Запасной ход (лучший без учёта табу),
		// используется если все допустимые ходы табуированы
		fallbackFrom, fallbackTo := -1, -1
		fallbackCost := maxInt
		fallbackKey := uint64(0)
		fallbackJob := -1

		// Итерация по случайно сгенерированным соседям
		for k := 0; k < neighbors; k++ {
			from := s.Rng.Intn(n)
			to := s.Rng.Intn(n - 1)
			if to >= from {
				to++
			}

			job := curr[from]
			key := moveKey(job, from, to)

			// Формирование соседнего решения
			copy(cand, curr)
			switch s.Cfg.Neighborhood {
			case NeighborhoodInsert:
				applyInsert(cand, from, to)
			case NeighborhoodSwap:
				applySwap(cand, from, to)
			default:
				applyInsert(cand, from, to)
			}

			cost := eval.MustMakespan(cand)
			evals++

			// Обновление хода
			if cost < fallbackCost {
				fallbackCost = cost
				fallbackFrom, fallbackTo = from, to
				fallbackKey = key
				fallbackJob = job
			}

			isTabu := tabu.IsTabu(key, iter)
			aspiration := cost < bestCost // критерий аспирации

			// Табуированный ход пропускается,
			// если не выполняется критерий аспирации
			if isTabu && !aspiration {
				continue
			}

			if cost < bestMoveCost {
				bestMoveCost = cost
				bestMoveFrom, bestMoveTo = from, to
				bestMoveKey = key
				bestMoveJob = job
			}
		}

		// Выбор хода: сначала допустимый лучший,
		chosenFrom, chosenTo := bestMoveFrom, bestMoveTo
		chosenCost := bestMoveCost
		chosenKey := bestMoveKey
		chosenJob := bestMoveJob

		if chosenFrom < 0 {
			chosenFrom, chosenTo = fallbackFrom, fallbackTo
			chosenCost = fallbackCost
			chosenKey = fallbackKey
			chosenJob = fallbackJob
		}

		// Нет допустимых ходов — завершаем поиск
		if chosenFrom < 0 {
			break
		}

		// Применение выбранного хода
		switch s.Cfg.Neighborhood {
		case NeighborhoodInsert:
			applyInsert(curr, chosenFrom, chosenTo)
		case NeighborhoodSwap:
			applySwap(curr, chosenFrom, chosenTo)
		default:
			applyInsert(curr, chosenFrom, chosenTo)
		}
		currCost = chosenCost

		// Добавление обратного хода в табу-список
		tenure := s.Cfg.TabuTenure
		if s.Cfg.TabuTenureRand > 0 {
			tenure += s.Rng.Intn(s.Cfg.TabuTenureRand + 1)
		}
		reverseKey := moveKey(chosenJob, chosenTo, chosenFrom)
		tabu.Add(reverseKey, iter+tenure)

		_ = chosenKey

		// Обновление глобально лучшего решения
		if currCost < bestCost {
			bestCost = currCost
			copy(best, curr)
		}
	}

	return opt.Result{
		Permutation: best,
		Makespan:    bestCost,
		Evaluations: evals,
		Iterations:  maxIter,
		Duration:    time.Since(start),
		Meta: map[string]any{
			"tabu_tenure":        s.Cfg.TabuTenure,
			"tabu_tenure_rand":   s.Cfg.TabuTenureRand,
			"neighbors_per_iter": s.Cfg.NeighborsPerIter,
			"neighborhood":       string(s.Cfg.Neighborhood),
		},
	}, nil
}

// tabuList — структура табу-списка.
// Реализована как кольцевой буфер фиксированного размера
// с map для быстрой проверки табуированности.
type tabuList struct {
	m   map[uint64]int // ключ → итерация истечения табу
	key []uint64       // кольцевой буфер ключей
	exp []int          // соответствующие сроки истечения
	i   int            // текущая позиция в кольце
}

// newTabuList создаёт табу-список заданной ёмкости.
func newTabuList(capacity int) *tabuList {
	if capacity < 8 {
		capacity = 8
	}
	return &tabuList{
		m:   make(map[uint64]int, capacity*2),
		key: make([]uint64, capacity),
		exp: make([]int, capacity),
		i:   0,
	}
}

// IsTabu проверяет, является ли ход табуированным на текущей итерации.
func (t *tabuList) IsTabu(k uint64, iter int) bool {
	if exp, ok := t.m[k]; ok && exp > iter {
		return true
	}
	return false
}

// Add добавляет новый табу-ход с указанием итерации истечения.
func (t *tabuList) Add(k uint64, expiry int) {
	// Удаление старого элемента из кольцевого буфера
	oldK := t.key[t.i]
	oldExp := t.exp[t.i]
	if oldK != 0 {
		if curExp, ok := t.m[oldK]; ok && curExp == oldExp {
			delete(t.m, oldK)
		}
	}

	t.key[t.i] = k
	t.exp[t.i] = expiry
	t.m[k] = expiry

	t.i++
	if t.i >= len(t.key) {
		t.i = 0
	}
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

// applySwap применяет swap-ход (обмен элементов в позициях i и j).
func applySwap(p []int, i, j int) {
	p[i], p[j] = p[j], p[i]
}

// applyInsert применяет insert-ход (элемент из позиции from вставляется в позицию to).
func applyInsert(p []int, from, to int) {
	if from == to {
		return
	}
	val := p[from]
	if from < to {
		copy(p[from:to], p[from+1:to+1])
		p[to] = val
		return
	}
	copy(p[to+1:from+1], p[to:from])
	p[to] = val
}

// moveKey формирует уникальный ключ хода
func moveKey(job, from, to int) uint64 {
	return (uint64(uint32(job)) << 42) |
		(uint64(uint32(from)) << 21) |
		uint64(uint32(to))
}

// max возвращает максимум из двух целых чисел.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
