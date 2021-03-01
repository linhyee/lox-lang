package lox

type LoxIterator interface {
	Len() int
	Add(item interface{})
	Get(index int) (interface{}, error)
	Set(index int, value interface{}) error
}
