package bench

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"flowShop/internal/flowshop"
	"flowShop/internal/opt"
)

type Algorithm struct {
	Name    string
	Factory func(seed int64) opt.Optimizer
}

type Case struct {
	Jobs         int
	Machines     int
	InstanceSeed int64
}

type Record struct {
	Algo     string
	Jobs     int
	Machines int
	Runs     int

	TimeBestMs float64
	TimeMeanMs float64
	TimeStdMs  float64

	MakespanBest int
	MakespanMean float64
	MakespanStd  float64
}

type Runner struct {
	Runs          int
	BaseSeed      int64
	PerRunTimeout time.Duration // 0 = no timeout
}

func (r Runner) RunCase(ctx context.Context, c Case, algo Algorithm) (Record, error) {
	instRng := randForSeed(c.InstanceSeed)
	inst := flowshop.RandomInstance(c.Jobs, c.Machines, 1, 99, instRng)

	makespans := make([]int, 0, r.Runs)
	timesMs := make([]float64, 0, r.Runs)

	for i := 0; i < r.Runs; i++ {
		runSeed := r.BaseSeed + int64(i)

		op := algo.Factory(runSeed)

		runCtx := ctx
		cancel := func() {}
		if r.PerRunTimeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, r.PerRunTimeout)
		}
		start := time.Now()
		res, err := op.Solve(runCtx, inst)
		dur := time.Since(start)
		cancel()

		if err != nil && runCtx.Err() != nil {
			return Record{}, fmt.Errorf("run %d: cancelled/timeout: %w", i, err)
		}
		if err != nil {
			return Record{}, fmt.Errorf("run %d: solve error: %w", i, err)
		}
		if len(res.Permutation) != inst.Jobs {
			return Record{}, fmt.Errorf("run %d: invalid permutation length %d (want %d)", i, len(res.Permutation), inst.Jobs)
		}

		makespans = append(makespans, res.Makespan)
		timesMs = append(timesMs, float64(dur.Microseconds())/1000.0)
	}

	msStats := CalcIntStats(makespans)
	tStats := CalcFloatStats(timesMs)

	return Record{
		Algo:     algo.Name,
		Jobs:     c.Jobs,
		Machines: c.Machines,
		Runs:     r.Runs,

		TimeBestMs: tStats.Best,
		TimeMeanMs: tStats.Mean,
		TimeStdMs:  tStats.Std,

		MakespanBest: msStats.Best,
		MakespanMean: msStats.Mean,
		MakespanStd:  msStats.Std,
	}, nil
}

func WriteCSV(path string, records []Record) error {
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"algo", "jobs", "machines", "runs",
		"time_best_ms", "time_mean_ms", "time_std_ms",
		"makespan_best", "makespan_mean", "makespan_std",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, r := range records {
		row := []string{
			r.Algo,
			itoa(r.Jobs),
			itoa(r.Machines),
			itoa(r.Runs),

			ftoa(r.TimeBestMs),
			ftoa(r.TimeMeanMs),
			ftoa(r.TimeStdMs),

			itoa(r.MakespanBest),
			ftoa(r.MakespanMean),
			ftoa(r.MakespanStd),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return w.Error()
}
