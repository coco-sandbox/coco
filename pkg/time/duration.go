// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package time

import (
	"time"
)

// Duration is a wrapper around time.Duration with additional utilities
type Duration time.Duration

// Milliseconds returns the duration as milliseconds
func (d Duration)Milliseconds() int64 {
	return time.Duration(d).Milliseconds()
}

// Microseconds returns the duration as microseconds
func (d Duration)Microseconds() int64 {
	return time.Duration(d).Microseconds()
}

// Nanoseconds returns the duration as nanoseconds
func (d Duration)Nanoseconds() int64 {
	return time.Duration(d).Nanoseconds()
}

// Seconds returns the duration as seconds (float64)
func (d Duration)Seconds() float64 {
	return time.Duration(d).Seconds()
}

// String implements fmt.Stringer
func (d Duration)String() string {
	return time.Duration(d).String()
}

// MarshalJSON implements json.Marshaler
func (d Duration)MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Duration(d).String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration)UnmarshalJSON(data []byte) error {
	str := string(data)
	if str == "null" {
		return nil
	}
	dur, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// Parse parses a duration string (e.g., "5s", "100ms", "1h30m")
func Parse(s string) (Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return Duration(d), nil
}

// MustParse parses a duration string and panics on error
func MustParse(s string) Duration {
	d, err := Parse(s)
	if err != nil {
		panic("invalid duration: " + s)
	}
	return d
}

// Common durations
var (
	Second      = Duration(time.Second)
	Minute      = Duration(time.Minute)
	Hour        = Duration(time.Hour)
	Millisecond = Duration(time.Millisecond)
	Microsecond = Duration(time.Microsecond)
)

// Since returns the duration since a time
func Since(t time.Time) Duration {
	return Duration(time.Since(t))
}

// Until returns the duration until a time
func Until(t time.Time) Duration {
	return Duration(time.Until(t))
}