package markdown

import (
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/russross/blackfriday/v2"

	"github.com/sysread/fnord/pkg/debug"
)

type tviewRenderer struct {
	style tcell.Style

	// Block quote rendering
	blockQuoteLevel int

	// List rendering
	nestingLevel int

	// Table rendering
	columnWidths []int
	tableHeader  []string
	currentRow   []string
	tableRows    [][]string
	isHeader     bool
}

func NewTviewRenderer() *tviewRenderer {
	return &tviewRenderer{
		style: tcell.StyleDefault.Normal(),
	}
}

func Render(md string) string {
	converted := blackfriday.Run([]byte(md), blackfriday.WithRenderer(NewTviewRenderer()))
	return string(converted)
}

func (r *tviewRenderer) Reset() *tviewRenderer {
	r.style = tcell.StyleDefault.Normal()
	return r
}

func (r *tviewRenderer) Bold() *tviewRenderer {
	r.style = r.style.Bold(true)
	return r
}

func (r *tviewRenderer) Italic() *tviewRenderer {
	// Italics are often not supported
	r.style = r.style.Underline(true)
	return r
}

func (r *tviewRenderer) Underline() *tviewRenderer {
	r.style = r.style.Underline(true)
	return r
}

func (r *tviewRenderer) StrikeThrough() *tviewRenderer {
	r.style = r.style.Reverse(true)
	return r
}

func (r *tviewRenderer) Dim() *tviewRenderer {
	r.style = r.style.Dim(true)
	return r
}

func (r *tviewRenderer) Code() *tviewRenderer {
	r.style = r.style.
		Foreground(tcell.ColorLemonChiffon)
	return r
}

func (r *tviewRenderer) Header() *tviewRenderer {
	r.style = r.style.
		Foreground(tcell.ColorLime).
		Bold(true).
		Underline(true)
	return r
}

func (r *tviewRenderer) BlockQuote() *tviewRenderer {
	r.style = r.style.Foreground(tcell.ColorPaleGoldenrod)
	return r
}

func (r *tviewRenderer) Link() *tviewRenderer {
	r.style = r.style.
		Foreground(tcell.ColorSkyblue).
		Underline(true)
	return r
}

func (r *tviewRenderer) Style(w io.Writer) *tviewRenderer {
	if r.style == tcell.StyleDefault {
		return r.Write(w, "[-:-:-]")
	} else {
		return r.Write(w, generateStyleTag(r.style))
	}
}

func (r *tviewRenderer) NewLine(w io.Writer) *tviewRenderer {
	return r.Write(w, "\n")
}

func (r *tviewRenderer) Bullet(w io.Writer) *tviewRenderer {
	return r.Write(w, "-")
}

func (r *tviewRenderer) Hash(w io.Writer) *tviewRenderer {
	return r.Write(w, "#")
}

func (r *tviewRenderer) Space(w io.Writer) *tviewRenderer {
	return r.Write(w, " ")
}

func (r *tviewRenderer) Write(w io.Writer, s string) *tviewRenderer {
	w.Write([]byte(s))
	return r
}

func (r *tviewRenderer) RenderHeader(w io.Writer, node *blackfriday.Node) {
}

func (r *tviewRenderer) RenderFooter(w io.Writer, node *blackfriday.Node) {
}

func (r *tviewRenderer) RenderNode(w io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	switch node.Type {
	case blackfriday.BlockQuote:
		if entering {
			r.blockQuoteLevel++
		} else {
			r.blockQuoteLevel--
		}

	case blackfriday.List:
		if entering {
			r.nestingLevel++ // Increase nesting level
		} else {
			r.nestingLevel-- // Decrease nesting level
			r.NewLine(w)
		}

	case blackfriday.Item:
		if entering {
			r.renderBullet(w).Space(w)
		}

	case blackfriday.Paragraph:
		if node.Parent.Type == blackfriday.Item {
			if !entering {
				r.NewLine(w)
			}
		} else {
			if entering {
				r.NewLine(w)
			} else {
				r.NewLine(w).NewLine(w)
			}
		}

	case blackfriday.Heading:
		if entering {
			r.renderHeader(w, node.Level)
		} else {
			r.resetStyle(w).NewLine(w)
		}

	case blackfriday.HorizontalRule:
		if entering {
			r.renderHorizontalRule(w)
		}

	case blackfriday.Emph:
		if entering {
			r.Italic().Style(w)
		} else {
			r.resetStyle(w)
		}

	case blackfriday.Strong:
		if entering {
			r.Bold().Style(w)
		} else {
			r.resetStyle(w)
		}

	case blackfriday.Del:
		if entering {
			r.StrikeThrough().Style(w)
		} else {
			r.resetStyle(w)
		}

	case blackfriday.Text:
		if r.blockQuoteLevel > 0 {
			r.renderBlockQuote(w, node.Literal)
		} else if node.Parent.Type != blackfriday.TableCell {
			r.renderText(w, node.Literal)
		}

	case blackfriday.CodeBlock:
		r.renderCodeBlock(w, node.Literal, string(node.Info))

	case blackfriday.Code:
		r.renderCode(w, node.Literal)

	case blackfriday.Table:
		if entering {
			r.startTable()
		} else {
			r.renderTable(w)
		}

	case blackfriday.TableHead:
		r.setInHeaderRow(entering)

	case blackfriday.TableRow:
		if entering {
			r.startRow()
		} else {
			r.finishRow()
		}

	case blackfriday.TableCell:
		if entering && node.FirstChild != nil {
			r.addTableCell(node.FirstChild.Literal)
		}

	case blackfriday.Link:
		if !entering {
			r.renderLink(w, node.Destination)
		}

	case blackfriday.HTMLBlock:
		r.renderCode(w, node.Literal)

	case blackfriday.HTMLSpan:
		r.renderCodeBlock(w, node.Literal, "html")

	case blackfriday.Softbreak:

	case blackfriday.Hardbreak:

	case blackfriday.Image:

	}

	return blackfriday.GoToNext
}

func (r *tviewRenderer) resetStyle(w io.Writer) *tviewRenderer {
	r.Reset().Style(w)
	return r
}

func (r *tviewRenderer) startTable() *tviewRenderer {
	r.columnWidths = nil
	r.tableRows = nil
	return r
}

func (r *tviewRenderer) setInHeaderRow(state bool) *tviewRenderer {
	r.isHeader = state
	return r
}

func (r *tviewRenderer) startRow() *tviewRenderer {
	r.currentRow = []string{}
	return r
}

func (r *tviewRenderer) finishRow() *tviewRenderer {
	// Store the row for later rendering
	if r.isHeader {
		r.tableHeader = r.currentRow
	} else {
		r.tableRows = append(r.tableRows, r.currentRow)
	}

	// Update column widths based on the current row
	for i, cell := range r.currentRow {
		if len(r.columnWidths) <= i {
			r.columnWidths = append(r.columnWidths, len(cell))
		} else if len(cell) > r.columnWidths[i] {
			r.columnWidths[i] = len(cell)
		}
	}

	// Reset the current row
	r.currentRow = nil

	return r
}

func (r *tviewRenderer) addTableCell(cell []byte) *tviewRenderer {
	r.currentRow = append(r.currentRow, string(cell))
	return r
}

func (r *tviewRenderer) renderTable(w io.Writer) *tviewRenderer {
	r.NewLine(w)

	// Render the table header
	if len(r.tableHeader) > 0 {
		for i, cell := range r.tableHeader {
			if i > 0 {
				r.Space(w).Write(w, "|").Space(w)
			}

			fmt.Fprintf(w, "%-*s", r.columnWidths[i], cell)
		}

		r.NewLine(w)

		// Render the separator row
		for i, width := range r.columnWidths {
			if i > 0 {
				r.Write(w, "-|-")
			}

			fmt.Fprintf(w, "%s", strings.Repeat("-", width))
		}

		r.NewLine(w)
	}

	// Render the table rows
	for _, row := range r.tableRows {
		for i, cell := range row {
			if i > 0 {
				r.Space(w).Write(w, "|").Space(w)
			}

			fmt.Fprintf(w, "%-*s", r.columnWidths[i], cell)
		}

		r.NewLine(w)
	}

	r.NewLine(w)

	r.columnWidths = nil
	r.tableRows = nil

	return r
}

func (r *tviewRenderer) renderBullet(w io.Writer) *tviewRenderer {
	for i := 0; i < r.nestingLevel; i++ {
		r.Space(w).Space(w) // Indent two spaces for each level of nesting
	}

	return r.Bullet(w)
}

func (r *tviewRenderer) renderLink(w io.Writer, uri []byte) *tviewRenderer {
	r.Write(w, " <").
		Link().Style(w).
		Write(w, string(uri)).
		resetStyle(w).
		Write(w, ">")
	return r
}

func (r *tviewRenderer) renderText(w io.Writer, text []byte) *tviewRenderer {
	r.Write(w, string(text))
	return r
}

func (r *tviewRenderer) renderHorizontalRule(w io.Writer) *tviewRenderer {
	r.Write(w, "\n────────────────────────────────────────────────────────────────────────────────\n")
	return r
}

func (r *tviewRenderer) renderBlockQuote(w io.Writer, text []byte) *tviewRenderer {
	r.BlockQuote().Style(w).Space(w).Space(w)

	for i := 0; i < r.blockQuoteLevel; i++ {
		r.Write(w, "|")
	}

	r.Write(w, " ").renderText(w, text).resetStyle(w)

	return r
}

func (r *tviewRenderer) renderHeader(w io.Writer, level int) *tviewRenderer {
	r.NewLine(w).Header().Style(w)

	for i := 0; i < level; i++ {
		r.Hash(w)
	}

	r.Space(w)
	return r
}

func (r *tviewRenderer) renderCode(w io.Writer, code []byte) *tviewRenderer {
	r.Code().Style(w).
		Write(w, "`").
		Write(w, string(code)).
		Write(w, "`").
		Reset().Style(w)

	return r
}

func (r *tviewRenderer) renderCodeBlock(w io.Writer, code []byte, language string) *tviewRenderer {
	formatted, err := highlightCode(string(code), language)
	if err != nil {
		debug.Log("Error highlighting code: %v", err)
		formatted = string(code)
	}

	// First write a subheader with the language info
	if language != "" {
		r.NewLine(w).
			Code().Style(w).
			// For SOME reason, using 2 spaces here is causing it to line up
			// with the 4-space-indented code returned by highlightCode.
			Write(w, fmt.Sprintf("  # vim: ft=%s", language)).
			Reset().Style(w).
			NewLine(w)
	}

	// Then write the code
	r.Write(w, formatted).
		Reset().Style(w).
		NewLine(w)

	return r
}

// highlightCode highlights code, first converting it to ANSI and then to tview
// format codes.
func highlightCode(code, language string) (string, error) {
	var buf strings.Builder

	err := quick.Highlight(&buf, code, language, "terminal16m", "monokai")
	if err != nil {
		return "", err
	}

	// Now indent it with 4 spaces
	var indented strings.Builder
	for _, line := range strings.Split(buf.String(), "\n") {
		indented.WriteString("    ")
		indented.WriteString(line)
		indented.WriteString("\n")
	}

	return tview.TranslateANSI(indented.String()), nil
}

// generateStyleTag creates a style tag from tcell.Style
func generateStyleTag(style tcell.Style) string {
	fg, bg, attr := style.Decompose()
	tag := ""

	if fg != tcell.ColorDefault {
		tag += fmt.Sprintf("[#%06x", fg.Hex())
	}
	if bg != tcell.ColorDefault {
		if tag == "" {
			tag += "["
		}
		tag += fmt.Sprintf(":#%06x", bg.Hex())
	}

	// Handle attributes
	attrTag := ":"
	if attr&tcell.AttrBold != 0 {
		attrTag += "b"
	}
	if attr&tcell.AttrUnderline != 0 {
		attrTag += "u"
	}
	if attr&tcell.AttrReverse != 0 {
		attrTag += "r"
	}
	if attr&tcell.AttrDim != 0 {
		attrTag += "d"
	}
	if attr&tcell.AttrBlink != 0 {
		attrTag += "i"
	}
	if attr&tcell.AttrItalic != 0 {
		attrTag += "i"
	}
	if attr&tcell.AttrStrikeThrough != 0 {
		attrTag += "s"
	}

	if attrTag != "" {
		if tag == "" {
			tag += "[::"
		} else {
			tag += "::"
		}
		tag += attrTag[1:]
	}

	if tag != "" {
		tag += "]"
	}

	return tag
}
