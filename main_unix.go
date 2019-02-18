//+build !windows

package main

import "golang.org/x/sys/unix"

func localExec(argv0 string, argv []string, envv []string) error {
	return unix.Exec(argv0, argv, envv)
}
