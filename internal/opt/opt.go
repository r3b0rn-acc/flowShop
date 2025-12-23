package opt

import (
	"context"
	"time"

	"flowShop/internal/flowshop"
)

type Optimizer interface {
	Solve(ctx context.Context, inst *flowshop.Instance) (Result, error)
}

type Result struct {
	Permutation []int
	Makespan    int
	Evaluations int
	Iterations  int
	Duration    time.Duration
	Meta        map[string]any
}
