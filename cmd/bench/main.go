package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"flowShop/internal/aco"
	"flowShop/internal/bench"
	"flowShop/internal/flowshop"
	"flowShop/internal/ga"
	"flowShop/internal/opt"
	"flowShop/internal/pso"
	"flowShop/internal/sa"
	"flowShop/internal/ts"
)

type gaAdapter struct{ s *ga.Solver }

func (a gaAdapter) Solve(ctx context.Context, inst *flowshop.Instance) (opt.Result, error) {
	return a.s.Solve(ctx, inst)
}

// Фабрики

func newGAFactory(cfg ga.Config) func(seed int64) opt.Optimizer {
	return func(seed int64) opt.Optimizer {
		solver, _ := ga.New(cfg, rand.New(rand.NewSource(seed)))
		return gaAdapter{s: solver}
	}
}

func newSAFactory(cfg sa.Config) func(seed int64) opt.Optimizer {
	return func(seed int64) opt.Optimizer {
		solver, _ := sa.New(cfg, rand.New(rand.NewSource(seed)))
		return solver
	}
}

func newTSFactory(cfg ts.Config) func(seed int64) opt.Optimizer {
	return func(seed int64) opt.Optimizer {
		solver, _ := ts.New(cfg, rand.New(rand.NewSource(seed)))
		return solver
	}
}

func newACOFactory(cfg aco.Config) func(seed int64) opt.Optimizer {
	return func(seed int64) opt.Optimizer {
		solver, _ := aco.New(cfg, rand.New(rand.NewSource(seed)))
		return solver
	}
}

func newPSOFactory(cfg pso.Config) func(seed int64) opt.Optimizer {
	return func(seed int64) opt.Optimizer {
		solver, _ := pso.New(cfg, rand.New(rand.NewSource(seed)))
		return solver
	}
}

func main() {
	// CLI флаги для настройки параметров алгоритмов и политики запуска
	var (
		out          = flag.String("out", "artifacts/results.csv", "путь к выходному CSV-файлу")
		pairs        = flag.String("pairs", "20x5,50x10,100x20", "конфигурации: количество работ Х количество станков (через запятую)")
		algos        = flag.String("algos", "GA,SA,TS,ACO,PSO", "список алгоритмов: GA, SA, TS, ACO, PSO (через запятую)")
		runs         = flag.Int("runs", 30, "количество запусков каждого алгоритма (с разными сидами)")
		baseSeed     = flag.Int64("seed", 1000, "базовый сид для запусков алгоритмов")
		instanceSeed = flag.Int64("instance_seed", 777, "базовый сид для генерации экземпляров задачи (фиксирован для конфигурации)")
		perRunTO     = flag.Duration("per_run_timeout", 0, "таймаут одного запуска; 0 — без ограничения")

		// --- Генетический алгоритм ---
		gaPop   = flag.Int("ga_pop", 150, "размер популяции")
		gaGen   = flag.Int("ga_gen", 400, "количество поколений")
		gaElite = flag.Int("ga_elite", 4, "размер элиты (количество лучших особей)")
		gaTour  = flag.Int("ga_tour", 5, "размер турнирной выборки")
		gaCx    = flag.Float64("ga_cx", 0.90, "вероятность применения кроссовера")
		gaMut   = flag.Float64("ga_mut", 0.15, "вероятность мутации")

		// --- Алгоритм имитации отжига ---
		saIterPerJob = flag.Int("sa_iter_per_job", 2500, "количество итераций на одну работу (используется, если sa_iter == 0)")
		saIter       = flag.Int("sa_iter", 0, "общее количество итераций (0 => sa_iter_per_job × nJobs)")
		saT0         = flag.Float64("sa_t0", 2000.0, "начальная температура")
		saTmin       = flag.Float64("sa_tmin", 0.5, "конечная температура")
		saAlpha      = flag.Float64("sa_alpha", 0.995, "коэффициент охлаждения (alpha)")
		saNeigh      = flag.String("sa_neigh", "swap", "тип окрестности: swap | insert")

		// --- Табу-поиск ---
		tsIterPerJob = flag.Int("ts_iter_per_job", 250, "количество итераций на одну работу (используется, если ts_iter == 0)")
		tsIter       = flag.Int("ts_iter", 0, "общее количество итераций (0 => ts_iter_per_job × nJobs)")
		tsTenure     = flag.Int("ts_tenure", 7, "длина табу-списка (в итерациях)")
		tsTenureRand = flag.Int("ts_tenure_rand", 3, "случайное добавление к сроку табу [0..rand]")
		tsNeighbors  = flag.Int("ts_neighbors", 90, "количество рассматриваемых соседей на итерацию")
		tsNeigh      = flag.String("ts_neigh", "insert", "тип окрестности: insert | swap")

		// --- Муравьиный алгоритм ---
		acoIterPerJob = flag.Int("aco_iter_per_job", 120, "количество итераций на одну работу (используется, если aco_iter == 0)")
		acoIter       = flag.Int("aco_iter", 0, "общее количество итераций (0 => aco_iter_per_job × nJobs)")
		acoAnts       = flag.Int("aco_ants", 35, "количество муравьёв")
		acoA          = flag.Float64("aco_alpha", 1.0, "коэффициент alpha (влияние феромонов)")
		acoB          = flag.Float64("aco_beta", 2.0, "коэффициент beta (влияние эвристики)")
		acoRho        = flag.Float64("aco_rho", 0.20, "коэффициент rho (испарения феромонов)")
		acoQ          = flag.Float64("aco_q", 1000.0, "константа отложения феромонов")
		acoTau0       = flag.Float64("aco_tau0", 1.0, "начальный уровень феромонов")
		acoCandK      = flag.Int("aco_k", 0, "размер списка кандидатов (0 — все оставшиеся)")

		// --- Рой частиц ---
		psoIterPerJob = flag.Int("pso_iter_per_job", 180, "количество итераций на одну работу (используется, если pso_iter == 0)")
		psoIter       = flag.Int("pso_iter", 0, "общее количество итераций (0 => pso_iter_per_job × nJobs)")
		psoParticles  = flag.Int("pso_particles", 60, "количество частиц")
		psoW          = flag.Float64("pso_w", 0.729, "коэффициент W (инерция)")
		psoC1         = flag.Float64("pso_c1", 1.49445, "коэффициент C1 (когнитивный)")
		psoC2         = flag.Float64("pso_c2", 1.49445, "коэффициент C2 (социальный)")
		psoVMax       = flag.Float64("pso_vmax", 0.25, "ограничение скорости частицы (<=0 — без ограничения)")
		psoPosMin     = flag.Float64("pso_pos_min", 0.0, "минимальное значение позиции частицы")
		psoPosMax     = flag.Float64("pso_pos_max", 1.0, "максимальное значение позиции частицы")
	)
	flag.Parse()

	ctx := context.Background()

	cases, err := parsePairs(*pairs, *instanceSeed)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Конфликт:", err)
		os.Exit(2)
	}

	gaCfg := ga.Config{
		Population:     *gaPop,
		Generations:    *gaGen,
		Elite:          *gaElite,
		TournamentSize: *gaTour,
		CrossoverRate:  *gaCx,
		MutationRate:   *gaMut,
	}
	if err := gaCfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "Конфликт в конфигурации генетического алгоритма:", err)
		os.Exit(2)
	}

	saCfg := sa.Config{
		Iterations:       *saIter,
		IterationsPerJob: *saIterPerJob,
		InitialTemp:      *saT0,
		FinalTemp:        *saTmin,
		Alpha:            *saAlpha,
		Neighborhood:     sa.Neighborhood(*saNeigh),
	}
	if err := saCfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "Конфликт в конфигурации алгоритма имитации отжига:", err)
		os.Exit(2)
	}

	tsCfg := ts.Config{
		Iterations:       *tsIter,
		IterationsPerJob: *tsIterPerJob,
		TabuTenure:       *tsTenure,
		TabuTenureRand:   *tsTenureRand,
		NeighborsPerIter: *tsNeighbors,
		Neighborhood:     ts.Neighborhood(*tsNeigh),
	}
	if err := tsCfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "Конфликт в конфигурации табушифтинга:", err)
		os.Exit(2)
	}

	acoCfg := aco.Config{
		Iterations:       *acoIter,
		IterationsPerJob: *acoIterPerJob,
		Ants:             *acoAnts,
		Alpha:            *acoA,
		Beta:             *acoB,
		Rho:              *acoRho,
		Q:                *acoQ,
		Tau0:             *acoTau0,
		CandidateK:       *acoCandK,
	}
	if err := acoCfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "Конфликт в конфигурации муравьиного алгоритма:", err)
		os.Exit(2)
	}

	psoCfg := pso.Config{
		Iterations:       *psoIter,
		IterationsPerJob: *psoIterPerJob,
		Particles:        *psoParticles,
		W:                *psoW,
		C1:               *psoC1,
		C2:               *psoC2,
		VMax:             *psoVMax,
		PosMin:           *psoPosMin,
		PosMax:           *psoPosMax,
	}
	if err := psoCfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "Конфликт в конфигурации роя частиц:", err)
		os.Exit(2)
	}

	available := map[string]bench.Algorithm{
		"GA":  {Name: "GA", Factory: newGAFactory(gaCfg)},
		"SA":  {Name: "SA", Factory: newSAFactory(saCfg)},
		"TS":  {Name: "TS", Factory: newTSFactory(tsCfg)},
		"ACO": {Name: "ACO", Factory: newACOFactory(acoCfg)},
		"PSO": {Name: "PSO", Factory: newPSOFactory(psoCfg)},
	}

	var selected []bench.Algorithm
	for _, a := range splitCSV(*algos) {
		al, ok := available[a]
		if !ok {
			fmt.Fprintf(os.Stderr, "Алгоритм не предоставлен в программе %q; доступные: %v\n", a, keys(available))
			os.Exit(2)
		}
		selected = append(selected, al)
	}

	runner := bench.Runner{
		Runs:          *runs,
		BaseSeed:      *baseSeed,
		PerRunTimeout: *perRunTO,
	}

	var records []bench.Record
	for _, c := range cases {
		for _, a := range selected {
			fmt.Printf("Запущен алгоритм %s; %d работ %d машин (общее кол-во запусков=%d)...\n", a.Name, c.Jobs, c.Machines, runner.Runs)

			rec, err := runner.RunCase(ctx, c, a)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Ошибка:", err)
				os.Exit(1)
			}
			records = append(records, rec)

			fmt.Printf("  Значение целевой функции: лучшее=%d среднее=%.2f стандартное отклонение=%.2f | Время: среднее=%.2fms среднее отклонение=%.2fms\n",
				rec.MakespanBest, rec.MakespanMean, rec.MakespanStd,
				rec.TimeMeanMs, rec.TimeStdMs,
			)
		}
	}

	if err := bench.WriteCSV(*out, records); err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка при записи в CSV:", err)
		os.Exit(1)
	}
	fmt.Println("Saved:", *out)
}

// helpers

func parsePairs(s string, baseInstanceSeed int64) ([]bench.Case, error) {
	parts := splitCSV(s)
	cases := make([]bench.Case, 0, len(parts))

	for i, p := range parts {
		jm := strings.Split(p, "x")
		if len(jm) != 2 {
			return nil, fmt.Errorf("пара %q невалидной схемы, пример: 50x10", p)
		}
		jobs, err := atoiStrict(jm[0])
		if err != nil {
			return nil, fmt.Errorf("пара %q: ошибка парсинга количества работ: %w", p, err)
		}
		machines, err := atoiStrict(jm[1])
		if err != nil {
			return nil, fmt.Errorf("пара %q: ошибка парсинга количества машин: %w", p, err)
		}
		if jobs <= 0 || machines <= 0 {
			return nil, fmt.Errorf("пара %q: количество работ и машин должно быть > 0", p)
		}

		seed := baseInstanceSeed + int64(i)*10_000 + int64(jobs)*100 + int64(machines)

		cases = append(cases, bench.Case{
			Jobs:         jobs,
			Machines:     machines,
			InstanceSeed: seed,
		})
	}

	return cases, nil
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func atoiStrict(s string) (int, error) {
	s = strings.TrimSpace(s)
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func keys(m map[string]bench.Algorithm) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
