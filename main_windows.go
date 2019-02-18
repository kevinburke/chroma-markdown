//+build windows

package main

import "golang.org/x/sys/windows"

func localExec(argv0 string, argv []string, envv []string) error {
	return windows.Exec(argv0, argv, envv)
}
