package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// dataPipedIn returns true if the user piped data via stdin.
func dataPipedIn() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func checkError(err error, msg string) {
	if err != nil {
		if msg != "" {
			fmt.Fprintf(os.Stderr, "Error %s: %v\n", msg, err)
		} else {
			fmt.Fprintf(os.Stderr, "Encountered an error: %v", err)
		}
		os.Exit(2)
	}
}

func init() {
	flag.Usage = func() {
		os.Stderr.WriteString(`usage: chroma-markdown [markdown-file]

`)
		flag.PrintDefaults()
	}
}

func highlight(w io.Writer, source, lexer, style string) error {
	// Determine lexer.
	l := lexers.Get(lexer)
	if l == nil {
		l = lexers.Analyse(source)
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)

	// Determine formatter.
	f := html.New(html.WithClasses(true), html.TabWidth(4))

	// Determine style.
	s := styles.Get(style)
	if s == nil {
		s = styles.Fallback
	}

	it, err := l.Tokenise(nil, source)
	if err != nil {
		return err
	}
	return f.Format(w, s, it)
}

// openInput picks the markdown source. An explicit file argument always wins:
// stdin is only read when no argument is given (or the argument is "-"), since
// under make/cron/CI stdin is often an inherited pipe that is never closed.
func openInput(args []string, stdin *os.File) (io.ReadCloser, error) {
	switch {
	case len(args) == 1 && args[0] != "-":
		return os.Open(args[0])
	case len(args) == 0 && dataPipedIn():
		return io.NopCloser(stdin), nil
	case len(args) == 1:
		return io.NopCloser(stdin), nil
	default:
		return nil, errUsage
	}
}

var errUsage = errors.New("expected exactly one markdown file (or \"-\" for stdin)")

func main() {
	css := flag.String("css", "", "Path to a CSS import to include at the beginning of the output")
	style := flag.String("style", "native", "CSS style to use")
	version := flag.Bool("version", false, "Print the version string")
	v := flag.Bool("v", false, "Print the version string")
	flag.Parse()
	if *version || *v {
		fmt.Printf("chroma-markdown version %s\n", Version)
		return
	}
	in, err := openInput(flag.Args(), os.Stdin)
	if err != nil {
		if err == errUsage {
			fmt.Fprintf(os.Stderr, "chroma-markdown: %v\n", err)
			flag.Usage()
			os.Exit(2)
		}
		checkError(err, "opening file")
	}
	defer in.Close()
	out := new(bytes.Buffer)
	currentCodeBlock := new(bytes.Buffer)
	started := false
	bs := bufio.NewScanner(in)
	lang := ""
	needCSS := false
	for bs.Scan() {
		text := bs.Text()
		trimmed := strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, "```") {
			if started {
				// TODO: compile the code block to markdown
				quickErr := highlight(out, currentCodeBlock.String(), lang, *style)
				checkError(quickErr, "highlighting source code")
				started = false
				currentCodeBlock.Reset()
				lang = ""
				needCSS = true
				continue
			}
			lang = trimmed[3:]
			started = true
			continue
		}
		if started {
			currentCodeBlock.WriteString(text)
			currentCodeBlock.WriteByte('\n')
		} else {
			out.WriteString(text)
			out.WriteByte('\n')
		}
	}
	checkError(bs.Err(), "reading markdown file")
	if started {
		fmt.Fprintln(os.Stderr, "chroma-markdown: warning: unterminated code block")
	}

	input := new(bytes.Buffer)
	if needCSS && *css != "" {
		fmt.Fprintf(input, "<link rel=\"stylesheet\" type=\"text/css\" href=\"%s\" />\n", *css)
	}
	input.Write(out.Bytes())

	// Shell out to markdown because of
	// https://github.com/russross/blackfriday/issues/403. The input goes over
	// the child's stdin (no temp file to leak) and the child inherits our
	// environment, so PATH/LANG etc. are preserved.
	bin, args := "cmark", []string{"--unsafe"}
	cmark, lookErr := exec.LookPath(bin)
	if lookErr != nil {
		bin, args = "markdown", nil
		cmark, lookErr = exec.LookPath(bin)
		checkError(lookErr, "finding markdown binary")
	}
	cmd := exec.Command(cmark, args...)
	cmd.Stdin = input
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	checkError(cmd.Run(), "running "+bin)
}
