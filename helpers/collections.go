package helpers

func SliceContains[T comparable, S []T](slice S, item T) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}

func RemoveFromSlice[T comparable, S ~[]T](slice S, item T) (result S) {
	for _, i := range slice {
		if i == item {
			continue
		}
		result = append(result, i)
	}
	return
}

func GetKeysFromMap[K comparable, V comparable, M map[K]V, R []K](v M) R {
	result := make(R, len(v))
	for i := range v {
		result = append(result, i)
	}
	return result
}
