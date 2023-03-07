package main

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64 | ~string
}

func Max[T Ordered](x, y T) T {
	if x > y {
		return x
	}
	return y
}

func Min[T Ordered](x, y T) T {
	if x < y {
		return x
	}

	return y
}

func Clip[T Ordered](a, min, max T) T {
	return Min(Max(a, min), max)
}
