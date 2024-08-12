package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Scrollable struct {
	*tview.Box

	app          *tview.Application
	children     []tview.Primitive
	renderHeight int
	scrollOffset int
	autoscroll   bool
}

// NewScrollable creates a new Scrollable widget.
func NewScrollable(app *tview.Application) *Scrollable {
	return &Scrollable{
		Box:          tview.NewBox(),
		app:          app,
		children:     []tview.Primitive{},
		renderHeight: 0,
		scrollOffset: 0,
		autoscroll:   false,
	}
}

// AddChild adds a new child widget to the scrollable area.
func (s *Scrollable) AddChild(child tview.Primitive) {
	s.children = append(s.children, child)

	if s.autoscroll {
		s.SetScrollOffset(s.scrollOffset + getPrimitiveHeight(child))
	}
}

// SetAutoscroll enables or disables autoscrolling.
func (s *Scrollable) SetAutoscroll(isEnabled bool) *Scrollable {
	s.autoscroll = isEnabled
	return s
}

// Draw renders the Scrollable and its children.
func (s *Scrollable) Draw(screen tcell.Screen) {
	s.DrawForSubclass(screen, s)

	// The scroll offset should be clamped to the maximum scroll offset. This
	// calculation can be incorrect when calculated before the widget is
	// initially rendered, so we recalculate it here to ensure it is properly
	// clamped.
	s.SetScrollOffset(s.scrollOffset)

	// Get available drawing area
	x, y, width, height := s.GetInnerRect()

	// Update the scrollable's render height
	s.renderHeight = height

	// Draw each child widget, adjusting for the scrollOffset
	currentY := y - s.scrollOffset

	for _, child := range s.children {
		childHeight := getPrimitiveHeight(child)

		if currentY+childHeight > y && currentY < y+height {
			child.SetRect(x, currentY, width, childHeight)
			child.Draw(screen)
		}

		currentY += childHeight
	}

	// The screen dimensions may have changed since the last render, so we
	// need to recalculate our render height.
	s.renderHeight = height
}

// InputHandler handles scrolling input.
func (s *Scrollable) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		switch event.Key() {
		case tcell.KeyUp:
			s.DecScrollOffset()
		case tcell.KeyDown:
			s.IncScrollOffset()
		case tcell.KeyPgUp:
			s.PageUp()
		case tcell.KeyPgDn:
			s.PageDown()
		case tcell.KeyHome:
			s.ScrollToBeginning()
		case tcell.KeyEnd:
			s.ScrollToEnd()
		default:
			switch event.Rune() {
			case 'j':
				s.IncScrollOffset()
			case 'k':
				s.DecScrollOffset()
			}
		}
	}
}

func (s *Scrollable) IncScrollOffset() {
	s.SetScrollOffset(s.scrollOffset + 1)
}

func (s *Scrollable) DecScrollOffset() {
	s.SetScrollOffset(s.scrollOffset - 1)
}

func (s *Scrollable) SetScrollOffset(offset int) {
	maxOffset := s.getMaxScrollOffset()

	if offset < 0 {
		offset = 0
	}

	if offset > maxOffset {
		offset = maxOffset
	}

	s.scrollOffset = offset
	s.SetAutoscroll(s.scrollOffset == maxOffset)
}

// PageUp scrolls the content up by one page.
func (s *Scrollable) PageUp() {
	s.SetScrollOffset((s.scrollOffset - s.renderHeight) / 2)
}

// PageDown scrolls the content down by one page.
func (s *Scrollable) PageDown() {
	s.SetScrollOffset((s.scrollOffset + s.renderHeight) / 2)
}

// ScrollToEnd scrolls the content to the end, ensuring the last visible page
// of content is shown.
func (s *Scrollable) ScrollToEnd() {
	s.scrollOffset = s.getMaxScrollOffset()
}

// ScrollToBeginning scrolls the content to the beginning.
func (s *Scrollable) ScrollToBeginning() {
	s.scrollOffset = 0
}

// getContentHeight calculates the total height of all children.
func (s *Scrollable) getContentHeight() int {
	height := 0

	for _, child := range s.children {
		height += getPrimitiveHeight(child)
	}

	return height
}

// getMaxScrollOffset calculates the maximum scroll offset.
func (s *Scrollable) getMaxScrollOffset() int {
	return s.getContentHeight() - s.renderHeight
}

// getPrimitiveHeight calculates the height of a primitive based on its content.
func getPrimitiveHeight(p tview.Primitive) int {
	_, _, _, height := p.GetRect()
	return height
}
