package maz

import "fmt"

// Options map type to facilitate calling functions with many variables.
type Options struct {
	options map[string]interface{}
}

// Constructor to initialize an options map
func NewOptions() *Options {
	return &Options{
		options: make(map[string]interface{}),
	}
}

// Sets values in an options map.
func (a *Options) Set(key string, value interface{}) *Options {
	a.options[key] = value
	return a // Return the object for chaining
}

// Gets a value of any type from the options map.
func (a *Options) Get(key string) (interface{}, bool) {
	value, ok := a.options[key]
	return value, ok
}

// Gets string value in an options map.
func (a *Options) GetString(key string) (string, bool) {
	value, ok := a.options[key]
	if !ok {
		return "", false // If not set, return empty string
	}
	strValue, ok := value.(string)
	return strValue, ok
}

// Gets boolean value in an options map.
func (a *Options) GetBool(key string) (bool, bool) {
	value, ok := a.options[key]
	if !ok {
		return false, false // If not set, return false
	}
	boolValue, ok := value.(bool)
	return boolValue, ok
}

// Gets integer value in an options map.
func (a *Options) GetInt(key string) (int, bool) {
	value, ok := a.options[key]
	if !ok {
		return 0, false // If not set, return 0
	}
	intValue, ok := value.(int)
	return intValue, ok
}

// Validate required keys.
func (a *Options) Validate(requiredKeys []string) error {
	for _, key := range requiredKeys {
		if _, ok := a.options[key]; !ok {
			return fmt.Errorf("missing required parameter: %s", key)
		}
	}
	return nil
}

// Returns the number of entries in the set of options.
func (a *Options) Count() int {
	return len(a.options)
}
