package bench

import (
	"math/rand"
	"path/filepath"
	"strconv"
)

func randForSeed(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

func dirOf(path string) string {
	d := filepath.Dir(path)
	if d == "." {
		return ""
	}
	return d
}

func itoa(v int) string { return strconv.Itoa(v) }

func ftoa(v float64) string {
	return strconv.FormatFloat(v, 'f', 6, 64)
}
