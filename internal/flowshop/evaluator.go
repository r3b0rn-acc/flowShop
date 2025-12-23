package flowshop

import "fmt"

type Evaluator struct {
	inst              *Instance
	machineCompletion []int
}

func NewEvaluator(inst *Instance) (*Evaluator, error) {
	if err := inst.Validate(); err != nil {
		return nil, err
	}
	return &Evaluator{inst: inst, machineCompletion: make([]int, inst.Machines)}, nil
}

func (e *Evaluator) Makespan(perm []int) (int, error) {
	if e == nil || e.inst == nil {
		return 0, fmt.Errorf("nil evaluator")
	}
	if len(perm) != e.inst.Jobs {
		return 0, fmt.Errorf("permutation length must be %d (got %d)", e.inst.Jobs, len(perm))
	}
	if err := ValidatePermutation(perm, e.inst.Jobs); err != nil {
		return 0, err
	}

	for m := range e.machineCompletion {
		e.machineCompletion[m] = 0
	}

	for _, job := range perm {
		e.machineCompletion[0] += e.inst.Time(job, 0)
		for m := 1; m < e.inst.Machines; m++ {
			left := e.machineCompletion[m-1]
			up := e.machineCompletion[m]
			if left > up {
				e.machineCompletion[m] = left + e.inst.Time(job, m)
			} else {
				e.machineCompletion[m] = up + e.inst.Time(job, m)
			}
		}
	}
	return e.machineCompletion[e.inst.Machines-1], nil
}

func (e *Evaluator) MustMakespan(perm []int) int {
	ms, err := e.Makespan(perm)
	if err != nil {
		panic(err)
	}
	return ms
}
