// Copyright 2026 Matthew Wilson and Synesis Information Systems. All
// rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Created: 7th September 2026
 * Updated: 7th September 2026
 */

package main

import (
	"github.com/stretchr/testify/require"

	"testing"
	"time"
)

func Test_formatDuration(t *testing.T) {

	testCases := []struct {
		expected string
		input    time.Duration
		name     string
	}{
		{
			expected: "—",
			input:    0,
			name:     "zero",
		},
		{
			expected: "1 ns",
			input:    1 * time.Nanosecond,
			name:     "nanoseconds single",
		},
		{
			expected: "500 ns",
			input:    500 * time.Nanosecond,
			name:     "nanoseconds multiple",
		},
		{
			expected: "999 ns",
			input:    999 * time.Nanosecond,
			name:     "nanoseconds upper bound",
		},
		{
			expected: "1.00 µs",
			input:    1 * time.Microsecond,
			name:     "microseconds boundary",
		},
		{
			expected: "1.50 µs",
			input:    1500 * time.Nanosecond,
			name:     "microseconds fractional",
		},
		{
			expected: "42.50 µs",
			input:    42500 * time.Nanosecond,
			name:     "microseconds multiple",
		},
		{
			expected: "999.90 µs",
			input:    999*time.Microsecond + 900*time.Nanosecond,
			name:     "microseconds upper bound",
		},
		{
			expected: "1.00 ms",
			input:    1 * time.Millisecond,
			name:     "milliseconds boundary",
		},
		{
			expected: "1.50 ms",
			input:    1500 * time.Microsecond,
			name:     "milliseconds fractional",
		},
		{
			expected: "42.50 ms",
			input:    42500 * time.Microsecond,
			name:     "milliseconds multiple",
		},
		{
			expected: "999.00 ms",
			input:    999 * time.Millisecond,
			name:     "milliseconds upper bound",
		},
		{
			expected: "1.00 s",
			input:    1 * time.Second,
			name:     "seconds boundary",
		},
		{
			expected: "2.50 s",
			input:    2500 * time.Millisecond,
			name:     "seconds fractional",
		},
		{
			expected: "65.00 s",
			input:    65 * time.Second,
			name:     "seconds multiple",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := formatDuration(tc.input)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func Test_formatRate(t *testing.T) {

	testCases := []struct {
		expected string
		input    uint64
		name     string
	}{
		{
			expected: "—",
			input:    0,
			name:     "zero",
		},
		{
			expected: "1 /s",
			input:    1,
			name:     "single entry",
		},
		{
			expected: "999 /s",
			input:    999,
			name:     "hundreds",
		},
		{
			expected: "1,000 /s",
			input:    1000,
			name:     "thousands",
		},
		{
			expected: "12,345 /s",
			input:    12345,
			name:     "ten thousands",
		},
		{
			expected: "1,234,567 /s",
			input:    1234567,
			name:     "millions",
		},
		{
			expected: "1,000,000,000 /s",
			input:    1000000000,
			name:     "billions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := formatRate(tc.input)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func Test_formatThroughput(t *testing.T) {

	testCases := []struct {
		expected string
		input    uint64
		name     string
	}{
		{
			expected: "—",
			input:    0,
			name:     "zero",
		},
		{
			expected: "1 B/s",
			input:    1,
			name:     "single byte",
		},
		{
			expected: "512 B/s",
			input:    512,
			name:     "half kibibyte",
		},
		{
			expected: "1023 B/s",
			input:    1023,
			name:     "bytes upper bound",
		},
		{
			expected: "1.00 KiB/s",
			input:    1024,
			name:     "kibibytes boundary",
		},
		{
			expected: "1.50 KiB/s",
			input:    1536,
			name:     "kibibytes fractional",
		},
		{
			expected: "512.00 KiB/s",
			input:    512 * 1024,
			name:     "kibibytes multiple",
		},
		{
			expected: "1024.00 KiB/s",
			input:    1024*1024 - 1,
			name:     "kibibytes upper bound",
		},
		{
			expected: "1.00 MiB/s",
			input:    1024 * 1024,
			name:     "mebibytes boundary",
		},
		{
			expected: "1.50 MiB/s",
			input:    1024*1024 + 512*1024,
			name:     "mebibytes fractional",
		},
		{
			expected: "256.00 MiB/s",
			input:    256 * 1024 * 1024,
			name:     "mebibytes multiple",
		},
		{
			expected: "1024.00 MiB/s",
			input:    1024*1024*1024 - 1,
			name:     "mebibytes upper bound",
		},
		{
			expected: "1.00 GiB/s",
			input:    1024 * 1024 * 1024,
			name:     "gibibytes boundary",
		},
		{
			expected: "1.50 GiB/s",
			input:    1024*1024*1024 + 512*1024*1024,
			name:     "gibibytes fractional",
		},
		{
			expected: "5.00 GiB/s",
			input:    5 * 1024 * 1024 * 1024,
			name:     "gibibytes multiple",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := formatThroughput(tc.input)
			require.Equal(t, tc.expected, actual)
		})
	}
}

/* ///////////////////////////// end of file //////////////////////////// */
