package ga

import "flowShop/internal/opt"

func ToOptResult(bestPerm []int, bestMakespan, evals, gens int, meta map[string]any) opt.Result {
	permCopy := make([]int, len(bestPerm))
	copy(permCopy, bestPerm)
	return opt.Result{
		Permutation: permCopy,
		Makespan:    bestMakespan,
		Evaluations: evals,
		Iterations:  gens,
		Meta:        meta,
	}
}
