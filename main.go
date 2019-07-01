package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
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
	f := html.New(html.WithClasses(), html.TabWidth(4))

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

const Version = "0.0"

func main() {
	css := flag.String("css", "", "Path to a CSS import to include at the beginning of the output")
	style := flag.String("style", "native", "CSS style to use")
	version := flag.Bool("version", false, "Print the version string")
	v := flag.Bool("v", false, "Print the version string")
	flag.Parse()
	if *version || *v {
		fmt.Printf("chroma-markdown version %s\n", Version)
		os.Exit(2)
	}
	var r io.Reader
	if dataPipedIn() {
		r = os.Stdin
	} else {
		if flag.NArg() != 1 {
			flag.Usage()
		}
		file := flag.Arg(0)
		f, err := os.Open(file)
		checkError(err, "opening file")
		defer f.Close()
		r = bufio.NewReader(f)
	}
	out := new(bytes.Buffer)
	currentCodeBlock := new(bytes.Buffer)
	started := false
	bs := bufio.NewScanner(r)
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
	f, err := ioutil.TempFile("", "chroma-markdown-")
	checkError(err, "creating temporary file")
	w := bufio.NewWriter(f)
	if needCSS && *css != "" {
		_, writeErr := fmt.Fprintf(w, `<link rel="stylesheet" type="text/css" href="%s" />\n`, *css)
		checkError(writeErr, "writing data to temporary file")
	}
	_, writeErr := f.Write(out.Bytes())
	checkError(writeErr, "writing data to temporary file")
	// shell out to markdown because of
	// https://github.com/russross/blackfriday/issues/403
	cmark, lookErr := exec.LookPath("cmark")
	args := []string{cmark, "--unsafe", f.Name()}
	if lookErr != nil {
		cmark, lookErr = exec.LookPath("markdown")
		checkError(lookErr, "finding markdown binary")
		args = []string{cmark, f.Name()}
	}
	execErr := localExec(cmark, args, []string{})
	checkError(execErr, "executing markdown binary")
	if err := f.Close(); err != nil {
		checkError(err, "closing file")
	}
}
