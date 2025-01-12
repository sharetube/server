package ptrfrommap

func PtrFromMap[T any](m map[string]any, key string) *T {
	if v, ok := m[key]; ok {
		if val, ok := v.(T); ok {
			return &val
		}
	}
	return nil
}

func PtrFromStringMap(m map[string]string, key string) *string {
	if v, ok := m[key]; ok {
		return &v
	}
	return nil
}
