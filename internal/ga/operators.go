package ga

import "math/rand"

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

// tournamentSelect реализует турнирный отбор.
// возвращается индекс особи с наилучшим значением fitness (минимальное значение целевой функции).
func tournamentSelect(scores []int, tournamentSize int, rng *rand.Rand) int {
	best := rng.Intn(len(scores))
	bestScore := scores[best]
	for i := 1; i < tournamentSize; i++ {
		cand := rng.Intn(len(scores))
		if scores[cand] < bestScore {
			best = cand
			bestScore = scores[cand]
		}
	}
	return best
}

// orderCrossoverOX реализует оператор Order Crossover.
func orderCrossoverOX(
	p1, p2, c1, c2 []int,
	rng *rand.Rand,
	mark []int,
	stamp *int,
) {
	n := len(p1)

	// Выбор случайного отрезка [a, b)
	a := rng.Intn(n)
	b := rng.Intn(n)
	if a > b {
		a, b = b, a
	}
	if a == b {
		// Что бы длина сегмента не была 0
		b = (a + 1) % n
		if a > b {
			a, b = b, a
		}
	}

	// Инициализация потомков
	fill := func(dst []int) {
		for i := range dst {
			dst[i] = -1
		}
	}
	fill(c1)
	fill(c2)

	// Формирование первого потомка

	*stamp++
	curStamp := *stamp

	// Копирование сегмента из первого родителя
	for i := a; i < b; i++ {
		gene := p1[i]
		c1[i] = gene
		mark[gene] = curStamp
	}

	// Заполнение оставшихся позиций генами второго родителя
	pos := b % n
	for i := 0; i < n; i++ {
		gene := p2[(b+i)%n]
		if mark[gene] == curStamp {
			continue
		}
		for c1[pos] != -1 {
			pos = (pos + 1) % n
		}
		c1[pos] = gene
		mark[gene] = curStamp
	}

	// Формирование второго потомка

	*stamp++
	curStamp = *stamp

	for i := a; i < b; i++ {
		gene := p2[i]
		c2[i] = gene
		mark[gene] = curStamp
	}
	pos = b % n
	for i := 0; i < n; i++ {
		gene := p1[(b+i)%n]
		if mark[gene] == curStamp {
			continue
		}
		for c2[pos] != -1 {
			pos = (pos + 1) % n
		}
		c2[pos] = gene
		mark[gene] = curStamp
	}
}

// mutateSwap реализует оператор мутации Swap.
func mutateSwap(p []int, rng *rand.Rand) {
	if len(p) < 2 {
		return
	}
	i := rng.Intn(len(p))
	j := rng.Intn(len(p) - 1)
	if j >= i {
		j++
	}
	p[i], p[j] = p[j], p[i]
}
