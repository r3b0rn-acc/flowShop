package flowshop

import "fmt"

func ValidatePermutation(perm []int, n int) error {
	if len(perm) != n {
		return fmt.Errorf("permutation length must be %d (got %d)", n, len(perm))
	}
	seen := make([]bool, n)
	for i, v := range perm {
		if v < 0 || v >= n {
			return fmt.Errorf("perm[%d]=%d out of range [0,%d)", i, v, n)
		}
		if seen[v] {
			return fmt.Errorf("duplicate job id %d in permutation", v)
		}
		seen[v] = true
	}
	return nil
}
