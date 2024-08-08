package orderedmap

import "github.com/elliotchance/orderedmap/v2"

type KV[K, V comparable] struct {
	Key   K
	Value V
}

func OrderedMapFromArgs[K, V comparable](y []KV[K, V]) *orderedmap.OrderedMap[K, V] {
	x := orderedmap.NewOrderedMap[K, V]()
	for _, z := range y {
		x.Set(z.Key, z.Value)
	}
	return x
}
