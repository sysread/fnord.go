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

	// Table rendering
	columnWidths []int
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
	return r.Write(w, "•")
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
		r.NewLine(w)

	case blackfriday.Item:
		if entering {
			r.Bullet(w).Space(w)
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
			r.NewLine(w).Header().Style(w)

			for i := 0; i < node.Level; i++ {
				r.Hash(w)
			}

			r.Space(w)
		} else {
			r.Reset().Style(w).NewLine(w)
		}

	case blackfriday.HorizontalRule:
		if entering {
			r.Write(w, "\n────────────────────────────────────────────────────────────────────────────────\n")
		}

	case blackfriday.Emph:
		if entering {
			r.Italic().Style(w)
		} else {
			r.Reset().Style(w)
		}

	case blackfriday.Strong:
		if entering {
			r.Bold().Style(w)
		} else {
			r.Reset().Style(w)
		}

	case blackfriday.Del:
		if entering {
			r.StrikeThrough().Style(w)
		} else {
			r.Reset().Style(w)
		}

	case blackfriday.Text:
		if r.blockQuoteLevel > 0 {
			r.BlockQuote().Style(w).Write(w, "  ")

			for i := 0; i < r.blockQuoteLevel; i++ {
				r.Write(w, "|")
			}

			r.Write(w, " " + string(node.Literal)).
				Reset().Style(w)
		} else if node.Parent.Type != blackfriday.TableCell {
			r.Write(w, string(node.Literal))
		}

	case blackfriday.HTMLBlock:

	case blackfriday.CodeBlock:
		code := string(node.Literal)
		language := string(node.Info)

		formatted, err := highlightCode(code, language)
		if err != nil {
			debug.Log("Error highlighting code: %v", err)
			formatted = string(code)
		}

		// First write a subheader with the language info
		r.NewLine(w).
			Code().Style(w).
			// For SOME reason, using 2 spaces here is causing it to line up
			// with the 4-space-indented code returned by highlightCode.
			Write(w, fmt.Sprintf("  # vim: ft=%s", language)).
			Reset().Style(w).
			NewLine(w)

		// Then write the code
		r.Write(w, formatted).
			Reset().Style(w).
			NewLine(w)

	case blackfriday.Code:
		r.Code().
			Style(w).
			Write(w, "`").
			Write(w, string(node.Literal)).
			Write(w, "`").
			Reset().
			Style(w)

	case blackfriday.Table:
		if entering {
			r.columnWidths = nil // Reset column widths for a new table
			r.tableRows = nil    // Reset table rows for a new table
		} else {
			// Render the entire table after determining column widths
			for _, row := range r.tableRows {
				for i, cell := range row {
					if i > 0 {
						r.Space(w).Write(w, " | ").Space(w) // Separator between columns
					}
					fmt.Fprintf(w, "%-*s", r.columnWidths[i], cell) // Pad the cell
				}
				r.NewLine(w)
			}
			r.columnWidths = nil // Clear the state after the table is rendered
			r.tableRows = nil    // Clear the state after the table is rendered
		}

	case blackfriday.TableHead:
		r.isHeader = entering

	case blackfriday.TableRow:
		if entering {
			r.currentRow = []string{}
		} else {
			// Store the row for later rendering
			r.tableRows = append(r.tableRows, r.currentRow)

			// Update column widths based on the current row
			for i, cell := range r.currentRow {
				if len(r.columnWidths) <= i {
					r.columnWidths = append(r.columnWidths, len(cell))
				} else if len(cell) > r.columnWidths[i] {
					r.columnWidths[i] = len(cell)
				}
			}
			r.currentRow = nil
		}

	case blackfriday.TableCell:
		if entering {
			// Capture the cell content
			var cellContent string
			if node.FirstChild != nil {
				cellContent = string(node.FirstChild.Literal)
			}

			// Store the cell content for rendering later
			r.currentRow = append(r.currentRow, cellContent)
		}

	case blackfriday.Link:
		if !entering {
			uri := string(node.Destination)

			r.Write(w, " <").
				Link().Style(w).
				Write(w, uri).
				Reset().Style(w).
				Write(w, ">")
		}

	case blackfriday.HTMLSpan:

	case blackfriday.Softbreak:

	case blackfriday.Hardbreak:

	case blackfriday.Image:

	}

	return blackfriday.GoToNext
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
