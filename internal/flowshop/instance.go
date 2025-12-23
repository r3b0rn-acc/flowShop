package flowshop

import (
	"errors"
	"fmt"
	"math/rand"
)

type Instance struct {
	Jobs     int
	Machines int
	// ProcTimes length must be Jobs*Machines.
	ProcTimes []int
}

func NewInstance(jobs, machines int, procTimes []int) (*Instance, error) {
	inst := &Instance{Jobs: jobs, Machines: machines, ProcTimes: procTimes}
	if err := inst.Validate(); err != nil {
		return nil, err
	}
	return inst, nil
}

func (inst *Instance) Validate() error {
	if inst == nil {
		return errors.New("instance is nil")
	}
	if inst.Jobs <= 0 {
		return fmt.Errorf("jobs must be > 0 (got %d)", inst.Jobs)
	}
	if inst.Machines <= 0 {
		return fmt.Errorf("machines must be > 0 (got %d)", inst.Machines)
	}
	if len(inst.ProcTimes) != inst.Jobs*inst.Machines {
		return fmt.Errorf("procTimes length must be jobs*machines=%d (got %d)", inst.Jobs*inst.Machines, len(inst.ProcTimes))
	}
	for i, v := range inst.ProcTimes {
		if v < 0 {
			return fmt.Errorf("procTimes[%d] must be >= 0 (got %d)", i, v)
		}
	}
	return nil
}

func (inst *Instance) Time(job, machine int) int {
	return inst.ProcTimes[job*inst.Machines+machine]
}

func RandomInstance(jobs, machines, minTime, maxTime int, rng *rand.Rand) *Instance {
	if rng == nil {
		panic("генератор случайных чисел не инициализирован (nil)")
	}
	if minTime < 0 || maxTime < 0 || maxTime < minTime {
		panic("invalid time bounds")
	}
	pt := make([]int, jobs*machines)
	span := maxTime - minTime + 1
	for i := range pt {
		pt[i] = minTime
		if span > 1 {
			pt[i] += rng.Intn(span)
		}
	}
	inst, err := NewInstance(jobs, machines, pt)
	if err != nil {
		panic(err)
	}
	return inst
}
