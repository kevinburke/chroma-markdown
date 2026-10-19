package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "chroma-markdown-test-")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(dir, "chroma-markdown")
	if out, err := exec.Command("go", "build", "-trimpath", "-o", binPath, ".").CombinedOutput(); err != nil {
		os.Stderr.Write(out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func haveMarkdown() bool {
	for _, b := range []string{"cmark", "markdown"} {
		if _, err := exec.LookPath(b); err == nil {
			return true
		}
	}
	return false
}

const sample = "# hi\n\n```go\npackage main\n```\n"

type result struct {
	stdout, stderr string
	err            error
}

// run executes the binary with the given stdin. If stdin is nil, an open pipe
// that is never written to or closed is used, like an inherited stdin under
// make. It fails the test if the process doesn't exit promptly.
func run(t *testing.T, stdin *string, args ...string) result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if stdin != nil {
		cmd.Stdin = strings.NewReader(*stdin)
	} else {
		pr, pw, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer pr.Close()
		defer pw.Close()
		cmd.Stdin = pr
	}
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("timed out (hang); stderr: %s", stderr.String())
	}
	return result{stdout.String(), stderr.String(), err}
}

func writeSample(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "in.md")
	if err := os.WriteFile(p, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// A file argument must not hang when stdin is an open, silent pipe.
func TestFileArgIgnoresOpenStdin(t *testing.T) {
	if !haveMarkdown() {
		t.Skip("no cmark/markdown binary")
	}
	r := run(t, nil, writeSample(t))
	if r.err != nil {
		t.Fatalf("%v: %s", r.err, r.stderr)
	}
	if !strings.Contains(r.stdout, "<h1") {
		t.Errorf("missing heading in output: %q", r.stdout)
	}
}

func TestStdin(t *testing.T) {
	if !haveMarkdown() {
		t.Skip("no cmark/markdown binary")
	}
	in := sample
	for _, args := range [][]string{nil, {"-"}} {
		r := run(t, &in, args...)
		if r.err != nil {
			t.Fatalf("args %v: %v: %s", args, r.err, r.stderr)
		}
		if !strings.Contains(r.stdout, "<h1") {
			t.Errorf("args %v: missing heading: %q", args, r.stdout)
		}
	}
}

func TestCSSLinkEmitted(t *testing.T) {
	if !haveMarkdown() {
		t.Skip("no cmark/markdown binary")
	}
	r := run(t, nil, "-css", "style.css", writeSample(t))
	if r.err != nil {
		t.Fatalf("%v: %s", r.err, r.stderr)
	}
	if !strings.Contains(r.stdout, `href="style.css"`) {
		t.Errorf("css link missing: %q", r.stdout)
	}
	if strings.Contains(r.stdout, `\n`) {
		t.Errorf("literal backslash-n in output: %q", r.stdout)
	}
}

func TestNoArgsNoStdinIsUsageError(t *testing.T) {
	// With an open pipe on stdin and no args we can't tell the user wants
	// stdin, but dataPipedIn is true, so use two args to hit usage instead.
	r := run(t, nil, "a.md", "b.md")
	ee, ok := r.err.(*exec.ExitError)
	if !ok || ee.ExitCode() != 2 {
		t.Fatalf("want exit 2, got %v", r.err)
	}
	if !strings.Contains(r.stderr, "usage:") {
		t.Errorf("no usage in stderr: %q", r.stderr)
	}
}

func TestVersionExitsZero(t *testing.T) {
	r := run(t, nil, "-version")
	if r.err != nil {
		t.Fatalf("want exit 0, got %v", r.err)
	}
	if !strings.Contains(r.stdout, Version) {
		t.Errorf("stdout = %q", r.stdout)
	}
}

func TestMissingFileFails(t *testing.T) {
	r := run(t, nil, filepath.Join(t.TempDir(), "nope.md"))
	if r.err == nil {
		t.Fatal("want error for missing file")
	}
}

func TestNoTempFileLeak(t *testing.T) {
	if !haveMarkdown() {
		t.Skip("no cmark/markdown binary")
	}
	tmp := t.TempDir()
	in := writeSample(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binPath, in)
	cmd.Env = append(os.Environ(), "TMPDIR="+tmp)
	cmd.Stdin = strings.NewReader("")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	ents, _ := os.ReadDir(tmp)
	if len(ents) != 0 {
		t.Errorf("leaked %d temp files", len(ents))
	}
}

// A closed, empty stdin (e.g. `echo -n | chroma-markdown file.md`) used to
// make the tool read zero bytes from stdin, ignore the file, and exit 0 with
// empty output.
func TestFileArgIgnoresEmptyStdin(t *testing.T) {
	if !haveMarkdown() {
		t.Skip("no cmark/markdown binary")
	}
	empty := ""
	r := run(t, &empty, writeSample(t))
	if r.err != nil {
		t.Fatalf("%v: %s", r.err, r.stderr)
	}
	if !strings.Contains(r.stdout, "<h1") {
		t.Errorf("file contents not rendered: %q", r.stdout)
	}
}

// Same, with stdin redirected from a regular file with different content.
func TestFileArgIgnoresStdinFile(t *testing.T) {
	if !haveMarkdown() {
		t.Skip("no cmark/markdown binary")
	}
	other := filepath.Join(t.TempDir(), "other.md")
	if err := os.WriteFile(other, []byte("OTHER\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(other)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cmd := exec.Command(binPath, writeSample(t))
	cmd.Stdin = f
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "OTHER") || !strings.Contains(string(out), "<h1") {
		t.Errorf("read stdin instead of file arg: %q", out)
	}
}
