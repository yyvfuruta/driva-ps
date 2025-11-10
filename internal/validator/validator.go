// Package validator provides functions for validating data.
package validator

// Validator holds a map of validation errors.
type Validator struct {
	Errors map[string]string
}

func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// Valid returns true if the validator has no errors.
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// Check adds an error to the validator if a check is not "ok".
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// AddError adds an error to the validator if the key does not already exist.
func (v *Validator) AddError(key, message string) {
	_, exists := v.Errors[key]
	if !exists {
		v.Errors[key] = message
	}
}
