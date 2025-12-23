package pso

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"flowShop/internal/flowshop"
	"flowShop/internal/opt"
)

// Solver - структура реализации алгоритма роя частиц
type Solver struct {
	Cfg Config
	Rng *rand.Rand
}

// New возвращает новый PSO-солвер с валидацией конфигурации, с использованием инициализированного генератора случайных чисел.
// // Используется в фабриках.
func New(cfg Config, rng *rand.Rand) (*Solver, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if rng == nil {
		return nil, fmt.Errorf("генератор случайных чисел не инициализирован (nil)")
	}
	return &Solver{Cfg: cfg, Rng: rng}, nil
}

// particle описывает одну частицу роя.
type particle struct {
	// pos — позиция частицы
	pos []float64
	// vel — скорость частицы
	vel []float64

	// pBestPos — лучшая позиция частицы за всё время
	pBestPos []float64
	// pBestCost — значение целевой функции в pBestPos
	pBestCost int

	// Вспомогательные буферы
	permScratch []int
	idxScratch  []int
}

// Solve — реализация эвристики.
func (s *Solver) Solve(ctx context.Context, inst *flowshop.Instance) (opt.Result, error) {
	start := time.Now()

	// Валидация конфигурации
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

	iters := s.Cfg.Iterations
	if iters <= 0 {
		iters = s.Cfg.IterationsPerJob * n
	}

	// Инициализация частиц
	ps := make([]particle, s.Cfg.Particles)
	for i := range ps {
		ps[i] = particle{
			pos:         make([]float64, n),
			vel:         make([]float64, n),
			pBestPos:    make([]float64, n),
			pBestCost:   math.MaxInt,
			permScratch: make([]int, n),
			idxScratch:  make([]int, n),
		}
	}

	posMin, posMax := s.Cfg.PosMin, s.Cfg.PosMax
	doPosClamp := posMin < posMax

	// Случайная инициализация позиций и скоростей частиц
	for i := range ps {
		for d := 0; d < n; d++ {
			// Инициализация позиции
			if doPosClamp {
				ps[i].pos[d] = posMin + s.Rng.Float64()*(posMax-posMin)
			} else {
				ps[i].pos[d] = s.Rng.Float64()
			}
			// Инициализация скорости
			if s.Cfg.VMax > 0 {
				ps[i].vel[d] = (s.Rng.Float64()*2 - 1) * s.Cfg.VMax
			} else {
				ps[i].vel[d] = (s.Rng.Float64()*2 - 1) * 0.1
			}
		}

		// Оценка начального положения частицы
		decodeRandomKeys(ps[i].pos, ps[i].permScratch, ps[i].idxScratch)
		cost := eval.MustMakespan(ps[i].permScratch)

		ps[i].pBestCost = cost
		copy(ps[i].pBestPos, ps[i].pos)
	}

	evals := s.Cfg.Particles

	// Вычисление глобально лучшего решения
	gBestPos := make([]float64, n)
	gBestPerm := make([]int, n)
	gBestCost := math.MaxInt

	for i := range ps {
		if ps[i].pBestCost < gBestCost {
			gBestCost = ps[i].pBestCost
			copy(gBestPos, ps[i].pBestPos)
			decodeRandomKeys(gBestPos, gBestPerm, make([]int, n))
		}
	}

	w, c1, c2 := s.Cfg.W, s.Cfg.C1, s.Cfg.C2
	vMax := s.Cfg.VMax

	// Основной цикл
	for iter := 0; iter < iters; iter++ {
		// Для поддержки отмены через context
		if err := ctx.Err(); err != nil {
			return opt.Result{
				Permutation: gBestPerm,
				Makespan:    gBestCost,
				Evaluations: evals,
				Iterations:  iter,
				Duration:    time.Since(start),
				Meta: map[string]any{
					"stopped": "context",
				},
			}, err
		}

		for i := range ps {
			p := &ps[i]

			// Обновление скорости и позиции частицы
			for d := 0; d < n; d++ {
				r1 := s.Rng.Float64()
				r2 := s.Rng.Float64()

				v := w*p.vel[d] +
					c1*r1*(p.pBestPos[d]-p.pos[d]) +
					c2*r2*(gBestPos[d]-p.pos[d])

				// Ограничение скорости
				if vMax > 0 {
					if v > vMax {
						v = vMax
					} else if v < -vMax {
						v = -vMax
					}
				}
				p.vel[d] = v

				// Обновление позиции
				x := p.pos[d] + v
				if doPosClamp {
					if x < posMin {
						x = posMin
						p.vel[d] = 0
					} else if x > posMax {
						x = posMax
						p.vel[d] = 0
					}
				}
				p.pos[d] = x
			}

			// Оценка нового положения частицы
			decodeRandomKeys(p.pos, p.permScratch, p.idxScratch)
			cost := eval.MustMakespan(p.permScratch)
			evals++

			// Обновление личного лучшего решения
			if cost < p.pBestCost {
				p.pBestCost = cost
				copy(p.pBestPos, p.pos)
			}

			// Обновление глобального лучшего решения
			if cost < gBestCost {
				gBestCost = cost
				copy(gBestPos, p.pos)
				copy(gBestPerm, p.permScratch)
			}
		}
	}

	return opt.Result{
		Permutation: gBestPerm,
		Makespan:    gBestCost,
		Evaluations: evals,
		Iterations:  iters,
		Duration:    time.Since(start),
		Meta: map[string]any{
			"particles": s.Cfg.Particles,
			"w":         w,
			"c1":        c1,
			"c2":        c2,
			"vmax":      vMax,
			"pos_min":   posMin,
			"pos_max":   posMax,
		},
	}, nil
}

// decodeRandomKeys преобразует вещественные random-keys в перестановку,
func decodeRandomKeys(keys []float64, outPerm []int, idxScratch []int) {
	n := len(keys)
	for i := 0; i < n; i++ {
		idxScratch[i] = i
	}
	sort.Slice(idxScratch, func(i, j int) bool {
		a := idxScratch[i]
		b := idxScratch[j]
		ka := keys[a]
		kb := keys[b]
		if ka == kb {
			return a < b
		}
		return ka < kb
	})
	for i := 0; i < n; i++ {
		outPerm[i] = idxScratch[i]
	}
}
