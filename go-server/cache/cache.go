package cache

type Cache interface {
	Get(string, interface{}) error
	Set(string, interface{}) error
	Delete(string) error
	Flush() error
}
