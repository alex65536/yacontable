package goutil

func Filter[T any, F ~func(T) bool](items []T, f F) []T {
	cnt := 0
	for i := 0; i < len(items); i++ {
		if f(items[i]) {
			items[i], items[cnt] = items[cnt], items[i]
			cnt++
		}
	}
	return items[:cnt]
}

func FilterCopy[T any, F ~func(T) bool](items []T, f F) []T {
	res := make([]T, 0, len(items))
	for i := 0; i < len(items); i++ {
		if f(items[i]) {
			res = append(res, items[i])
		}
	}
	return res
}

func Map[T any, U any, F ~func(T) U](items []T, f F) []U {
	res := make([]U, len(items))
	for i, item := range items {
		res[i] = f(item)
	}
	return res
}

func MapWithErr[T any, U any, F ~func(T) (U, error)](items []T, f F) ([]U, error) {
	res := make([]U, len(items))
	for i, item := range items {
		var err error
		res[i], err = f(item)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}
