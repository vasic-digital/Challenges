package logging

// LogField creates a Field from a key-value pair. This is a
// convenience function for constructing structured log fields.
func LogField(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// StringField creates a Field with a string value.
func StringField(key, value string) Field {
	return Field{Key: key, Value: value}
}

// IntField creates a Field with an integer value.
func IntField(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64Field creates a Field with an int64 value.
func Int64Field(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64Field creates a Field with a float64 value.
func Float64Field(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// BoolField creates a Field with a boolean value.
func BoolField(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// ErrorField creates a Field for an error value. If err is nil,
// the value is set to the string "<nil>".
func ErrorField(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: "<nil>"}
	}
	return Field{Key: "error", Value: err.Error()}
}
