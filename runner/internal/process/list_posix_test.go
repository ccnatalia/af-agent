//go:build !windows

package process

import (
	"reflect"
	"testing"
)

func TestParseListOutput(t *testing.T) {
	got := parseListOutput([]byte("    1 /usr/lib/systemd/systemd --switched-root\n  123 /usr/bin/whoami\n  bad ignored\n  456\n"))
	want := []Info{
		{PID: 1, Command: "/usr/lib/systemd/systemd --switched-root"},
		{PID: 123, Command: "/usr/bin/whoami"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("processes = %#v, want %#v", got, want)
	}
}
