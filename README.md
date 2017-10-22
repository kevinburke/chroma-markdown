# chroma-markdown

This binary compiles Markdown files into HTML. In particular, code blocks of the
following format:

<pre><code># Section Title

```go
fmt.Println("hello world")
```</code></pre>

Will be parsed and compiled to HTML using [`chroma`, a tool for generating
syntax highlights][chroma]. The rest of the file will be compiled from markdown
to HTML.

```html
<h1>Section Title</h1>
<pre class="chroma"><span class="nx">fmt</span><span class="p">.</span><span class="nx">Println</span><span class="p">(</span><span class="s">&#34;hello world&#34;</span><span class="p">)</span>
</pre>
```

Input can be sent either on standard input or by specifying a filename at the
command line. All outputs are printed to standard output, where they can be
redirected to a file:

```html
cat index.md | chroma-markdown > output.html
```

#### Why?

You can use this with content management systems that want you to use an HTML
editor, like WordPress. There are tools that do Markdown compilation and tools
that do syntax highlighting but few that do both.

## Requirements

This requires either Commonmark (cmark) or Markdown (markdown) binaries to be
present on your `$PATH`. The Go markdown renderer has [a critical error][error]
that prevents in-memory Markdown compilation.

[error]: https://github.com/russross/blackfriday/issues/403
[chroma]: https://github.com/alecthomas/chroma

## Installation

Find your target operating system (darwin, windows, linux) and desired bin
directory, and modify the command below as appropriate:

    curl --silent --location --output=/usr/local/bin/chroma-markdown https://github.com/kevinburke/chroma-markdown/releases/download/0.1/chroma-markdown-linux-amd64 && chmod 755 /usr/local/bin/chroma-markdown

The latest version is 0.1.

If you have a Go development environment, you can also install via source code:

```
go get -u github.com/kevinburke/chroma-markdown
```

This should place a `chroma-markdown` binary in `$GOPATH/bin`, so for me,
`~/bin/chroma-markdown`.
