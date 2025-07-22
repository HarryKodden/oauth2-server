package utils

import "testing"

func TestGenerator(t *testing.T) {
	t.Run("hello world", func(t *testing.T) {
		got := "hello world"
		want := "hello world"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
