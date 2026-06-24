package clipboard

import "github.com/atotto/clipboard"

// Read returns the current contents of the clipboard.
func Read() (string, error) {
	return clipboard.ReadAll()
}

// Write sets the clipboard contents to the given string.
func Write(s string) error {
	return clipboard.WriteAll(s)
}
