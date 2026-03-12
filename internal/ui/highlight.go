package ui

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/rivo/tview"
)

// HighlightCode tokenizes code with chroma and converts to tview color tags.
func HighlightCode(code, language, syntaxStyle string) string {
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get(syntaxStyle)

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return tview.Escape(code)
	}

	var buf strings.Builder
	for _, token := range iterator.Tokens() {
		text := tview.Escape(token.Value)
		entry := style.Get(token.Type)
		if entry.Colour.IsSet() {
			hex := fmt.Sprintf("#%02x%02x%02x", entry.Colour.Red(), entry.Colour.Green(), entry.Colour.Blue())
			buf.WriteString("[" + hex + "]")
			buf.WriteString(text)
			buf.WriteString("[-]")
		} else {
			buf.WriteString(text)
		}
	}
	return buf.String()
}
