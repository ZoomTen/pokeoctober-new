package utils

type OrderedMap struct {
  keys   []interface{}
  values map[interface{}]interface{}
}

func NewOrderedMap() *OrderedMap {
  return &OrderedMap{
    keys:   []interface{}{},
    values: make(map[interface{}]interface{}),
  }
}

// Range mimics the All() iterator using a callback function.
// This was the standard pattern for iteration before Go 1.23.
func (om *OrderedMap) Range(fn func(key, value interface{}) bool) {
  for _, k := range om.keys {
    if !fn(k, om.values[k]) {
      break
    }
  }
}

func (om *OrderedMap) Set(key, value interface{}) {
  if _, exists := om.values[key]; !exists {
    om.keys = append(om.keys, key)
  }
  om.values[key] = value
}

func (om *OrderedMap) Get(key interface{}) (interface{}, bool) {
  val, exists := om.values[key]
  return val, exists
}

func (om *OrderedMap) Len() int {
  return len(om.keys)
}

func (om *OrderedMap) KeyAt(n int) (interface{}, bool) {
  if n < 0 || n >= len(om.keys) {
    return nil, false
  }
  return om.keys[n], true
}
