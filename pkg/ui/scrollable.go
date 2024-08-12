package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sysread/fnord/pkg/debug"
)

type Scrollable struct {
	*tview.Box

	app          *tview.Application
	children     []*tview.Frame
	renderHeight int
	scrollOffset int
	autoscroll   bool

	selectedChildIndex int
	focusedChildIndex  int
}

// NewScrollable creates a new Scrollable widget.
func NewScrollable(app *tview.Application) *Scrollable {
	return &Scrollable{
		Box:                tview.NewBox(),
		app:                app,
		children:           []*tview.Frame{},
		renderHeight:       0,
		scrollOffset:       0,
		autoscroll:         false,
		selectedChildIndex: -1,
		focusedChildIndex:  -1,
	}
}

func (s *Scrollable) ReFocus() {
	s.app.SetFocus(s)
	s.focusedChildIndex = -1
}

// getPrimitiveHeight calculates the height of a primitive based on its content.
func getPrimitiveHeight(p tview.Primitive) int {
	_, _, _, height := p.GetRect()
	return height
}

// AddChild adds a new child widget to the scrollable area.
func (s *Scrollable) AddChild(child tview.Primitive) {
	frame := tview.NewFrame(child).SetBorders(0, 0, 0, 0, 1, 1)
	s.children = append(s.children, frame)

	if s.autoscroll {
		s.SetScrollOffset(s.scrollOffset + getPrimitiveHeight(frame))
		s.selectedChildIndex = len(s.children) - 1
		s.scrollToChild()
	} else {
		s.NextChild()
	}
}

// SetAutoscroll enables or disables autoscrolling.
func (s *Scrollable) SetAutoscroll(isEnabled bool) *Scrollable {
	s.autoscroll = isEnabled
	return s
}

// Draw renders the Scrollable and its children.
func (s *Scrollable) Draw(screen tcell.Screen) {
	debug.Log("Scrollable drawn")
	s.DrawForSubclass(screen, s)

	s.SetScrollOffset(s.scrollOffset)

	x, y, width, height := s.GetInnerRect()
	s.renderHeight = height

	currentY := y - s.scrollOffset

	for i, child := range s.children {
		childHeight := getPrimitiveHeight(child)

		if currentY+childHeight > y && currentY < y+height {
			switch i {
			case s.focusedChildIndex:
				child.SetBorder(true).SetBorderColor(tcell.ColorGreen)
			case s.selectedChildIndex:
				child.SetBorder(true).SetBorderColor(tview.Styles.PrimaryTextColor)
			default:
				child.SetBorder(true).SetBorderColor(tview.Styles.PrimitiveBackgroundColor)
			}

			child.SetRect(x, currentY, width, childHeight)
			child.Draw(screen)
		}

		currentY += childHeight
	}
}

// InputHandler handles scrolling input.
func (s *Scrollable) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		debug.Log("Scrollable received event: %v", event)
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
		case tcell.KeyTab:
			s.NextChild()
		case tcell.KeyBacktab:
			s.PreviousChild()
		case tcell.KeyEnter:
			s.FocusSelectedChild(setFocus)
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

func (s *Scrollable) scrollToChild() {
	if len(s.children) == 0 {
		return
	}

	// Calculate the position of the selected child
	y := 0
	for i, child := range s.children {
		childHeight := getPrimitiveHeight(child)

		if i == s.selectedChildIndex {
			break
		}

		y += childHeight
	}

	// Calculate the visible area
	_, _, _, height := s.GetInnerRect()

	// Adjust scroll offset if the selected child is out of view
	if y < s.scrollOffset {
		s.SetScrollOffset(y)
	} else if y+getPrimitiveHeight(s.children[s.selectedChildIndex]) > s.scrollOffset+height {
		s.SetScrollOffset(y + getPrimitiveHeight(s.children[s.selectedChildIndex]) - height)
	}
}

func (s *Scrollable) NextChild() {
	s.selectedChildIndex = (s.selectedChildIndex + 1) % len(s.children)
	s.scrollToChild()
}

func (s *Scrollable) PreviousChild() {
	s.selectedChildIndex = (s.selectedChildIndex - 1 + len(s.children)) % len(s.children)
	s.scrollToChild()
}

func (s *Scrollable) FocusSelectedChild(setFocus func(p tview.Primitive)) {
	if len(s.children) > 0 {
		s.focusedChildIndex = s.selectedChildIndex
		frame := s.children[s.selectedChildIndex]
		setFocus(frame)
	}
}
