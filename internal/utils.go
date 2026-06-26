package internal

func CountInSlice[T comparable](s []T, e T) int {
	count := 0
	for _, el := range s {
		if el == e {
			count++
		}
	}

	return count
}

func MakeUniqueSlice[T comparable](s []T) []T {
	out := make([]T, 0)

	for _, q := range s {
		if CountInSlice(out, q) == 1 {
			continue
		}

		out = append(out, q)
	}

	return out
}
