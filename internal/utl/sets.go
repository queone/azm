package utl

// While Go doesn’t have a built-in 'Set' type, 'map' is a natural and efficient substitute.
// For better clarity, we wrap it in a custom type with helper methods that align with set
// operations. This keeps the code clean and self-explanatory while leveraging Go’s
// simplicity and performance. See protip https://que.one/golang/#using-struct-for-efficient-maps
type StringSet map[string]struct{}

// Create a new set

// Add an item to the set
func (s StringSet) Add(item string) {
	s[item] = struct{}{}
}

// Remove an item from the set
func (s StringSet) Remove(item string) {
	delete(s, item)
}

// Check if an item exists in the set
func (s StringSet) Exists(item string) bool {
	_, exists := s[item]
	return exists
}

// Get the size of the set
func (s StringSet) Size() int {
	return len(s)
}

// Other types of sets can be added in the future
