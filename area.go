// 14 march 2014

package ui

import (
	"fmt"
	"sync"
	"image"
)

// Area represents a blank canvas upon which programs may draw anything and receive arbitrary events from the user.
// An Area has an explicit size, represented in pixels, that may be different from the size shown in its Window; Areas have horizontal and vertical scrollbars that are hidden when not needed.
// The coordinate system of an Area always has an origin of (0,0) which maps to the top-left corner; all image.Points and image.Rectangles sent across Area's channels conform to this.
// The size of an Area must be at least 1x1 (that is, neither its width nor its height may be zero or negative).
// 
// To handle events to the Area, an Area must be paired with an AreaHandler.
// See AreaHandler for details.
// 
// Do not use an Area if you intend to read text.
// Area reads keys based on their position on a standard
// 101-key keyboard, and does no character processing.
// Character processing methods differ across operating
// systems; trying ot recreate these yourself is only going
// to lead to trouble.
// [Use TextArea instead, providing a TextAreaHandler.]
// 
// To facilitate development and debugging, for the time being, Areas only work on GTK+.
type Area struct {
	lock			sync.Mutex
	created		bool
	sysData		*sysData
	handler		AreaHandler
	initwidth		int
	initheight		int
}

// AreaHandler represents the events that an Area should respond to.
// You are responsible for the thread safety of any members of the actual type that implements ths interface.
// (Having to use this interface does not strike me as being particularly Go-like, but the nature of Paint makes channel-based event handling a non-option; in practice, deadlocks occur.)
type AreaHandler interface {
	// Paint is called when the Area needs to be redrawn.
	// The part of the Area that needs to be redrawn is stored in cliprect.
	// Before Paint() is called, this region is cleared with a system-defined background color.
	// You MUST handle this event, and you MUST return a valid image, otherwise deadlocks and panicking will occur.
	// The image returned must have the same size as rect (but does not have to have the same origin points).
	// Example:
	// 	imgFromFile, _, err := image.Decode(file)
	// 	if err != nil { panic(err) }
	// 	img := image.NewRGBA(imgFromFile.Rect)
	// 	draw.Draw(img, img.Rect, imgFromFile, image.ZP, draw.Over)
	// 	// ...
	// 	func (h *myAreaHandler) Paint(rect image.Rectangle) *image.RGBA {
	// 		return img.SubImage(rect).(*image.RGBA)
	// 	}
	Paint(cliprect image.Rectangle) *image.RGBA

	// Mouse is called when the Area receives a mouse event.
	// You are allowed to do nothing in this handler (to ignore mouse events).
	// See MouseEvent for details.
	// If repaint is true, the Area is marked as needing to be redrawn.
	Mouse(e MouseEvent) (repaint bool)

	// Key is called when the Area receives a keyboard event.
	// You are allowed to do nothing except return false for handled in this handler (to ignore keyboard events).
	// Do not do nothing but return true for handled; this may have unintended consequences.
	// See KeyEvent for details.
	// If repaint is true, the Area is marked as needing to be redrawn.
	Key(e KeyEvent) (handled bool, repaint bool)
}

// MouseEvent contains all the information for a mous event sent by Area.Mouse.
// Mouse button IDs start at 1, with 1 being the left mouse button, 2 being the middle mouse button, and 3 being the right mouse button.
// (TODO "If additional buttons are supported, they will be returned with 4 being the first additional button (XBUTTON1 on Windows), 5 being the second (XBUTTON2 on Windows), and so on."?) (TODO get the user-facing name for XBUTTON1/2; find out if there's a way to query available button count)
type MouseEvent struct {
	// Pos is the position of the mouse in the Area at the time of the event.
	// TODO rename to Pt or Point?
	Pos			image.Point

	// If the event was generated by a mouse button being pressed, Down contains the ID of that button.
	// Otherwise, Down contains 0.
	Down		uint

	// If the event was generated by a mouse button being released, Up contains the ID of that button.
	// Otherwise, Up contains 0.
	// If both Down and Up are 0, the event represents mouse movement (with optional held buttons; see below).
	// Down and Up shall not both be nonzero.
	Up			uint

	// If Down is nonzero, Count indicates the number of clicks: 1 for single-click, 2 for double-click.
	// If Count == 2, AT LEAST zero events with Count == 1 will have been sent prior.
	// (This is a platform-specific issue: some platforms send none, some send one, and some send two.)
	Count		uint

	// Modifiers is a bit mask indicating the modifier keys being held during the event.
	Modifiers		Modifiers

	// Held is a slice of button IDs that indicate which mouse buttons are being held during the event.
	// Held will not include Down and Up.
	// (TODO "There is no guarantee that Held is sorted."?)
	Held			[]uint
}

// HeldBits returns Held as a bit mask.
// Bit 0 maps to button 1, bit 1 maps to button 2, etc.
func (e MouseEvent) HeldBits() (h uintptr) {
	for _, x := range e.Held {
		h |= uintptr(1) << (x - 1)
	}
	return h
}

// A KeyEvent represents a keypress in an Area.
// 
// Key presses are based on their positions on a standard
// 101-key keyboard found on most computers. The
// names chosen for keys here are based on their names
// on US English QWERTY keyboards; see Key for details.
// 
// When you are finished processing the incoming event,
// return whether or not you did something in response
// to the given keystroke as the handled return of your
// AreaHandler's Key() implementation. If you send false,
// you indicate that you did not handle the keypress, and that
// the system should handle it instead. (Some systems will stop
// processing the keyboard event at all if you return true
// unconditionally, which may result in unwanted behavior like
// global task-switching keystrokes not being processed.)
// 
// Note that even given the above, some systems might intercept
// some keystrokes (like Alt-F4 on various Unix systems) before
// Area will ever see them (and the Area might get an incorrect
// KeyEvent in this case, but this is not guaranteed); be wary.
// 
// If a key is pressed that is not supported by Key, ExtKey,
// or Modifiers, no KeyEvent will be produced, and package
// ui will act as if false was returned for handled.
type KeyEvent struct {
	// Key is a byte representing a character pressed
	// in the typewriter section of the keyboard.
	// The value, which is independent of whether the
	// Shift key is held, is a constant with one of the
	// following (case-sensitive) values, drawn according
	// to the key's position on the keyboard.
	//    ` 1 2 3 4 5 6 7 8 9 0 - =
	//     q w e r t y u i o p [ ] \
	//      a s d f g h j k l ; '
	//       z x c v b n m , . /
	// The actual key entered will be the key at the respective
	// position on the user's keyboard, regardless of the actual
	// layout. (Some keyboards move \ to either the row above
	// or the row below but in roughly the same spot; this is
	// accounted for. Some keyboards have an additonal key
	// to the left of 'z' or additional keys to the right of '='; these
	// cannot be read.)
	// In addition, Key will contain
	// - ' ' (space) if the spacebar was pressed
	// - '\t' if Tab was pressed, regardless of Modifiers
	// - '\n' if the typewriter Enter key was pressed
	// - '\b' if the typewriter Backspace key was pressed
	// If this value is zero, see ExtKey.
	Key			byte

	// If Key is zero, ExtKey contains a predeclared identifier
	// naming an extended key. See ExtKey for details.
	// If both Key and ExtKey are zero, a Modifier by itself
	// was pressed. Key and ExtKey will not both be nonzero.
	ExtKey		ExtKey

	Modifiers		Modifiers

	// If Up is true, the key was released; if not, the key was pressed.
	// There is no guarantee that all pressed keys shall have
	// corresponding release events (for instance, if the user switches
	// programs while holding the key down, then releases the key).
	// Keys that have been held down are reported as multiple
	// key press events.
	Up			bool
}

// ExtKey represents keys that are not in the typewriter section of the keyboard.
type ExtKey uintptr
const (
	Escape ExtKey = iota + 1
	Insert
	Delete
	Home
	End
	PageUp
	PageDown
	Up
	Down
	Left
	Right
	F1			// F1..F12 are guaranteed to be consecutive
	F2
	F3
	F4
	F5
	F6
	F7
	F8
	F9
	F10
	F11
	F12
	N0			// numpad keys; independent of Num Lock state
	N1			// N0..N9 are guaranteed to be consecutive
	N2
	N3
	N4
	N5
	N6
	N7
	N8
	N9
	NDot
	NEnter
	NAdd
	NSubtract
	NMultiply
	NDivide
	_nextkeys		// for sanity check
)

// EffectiveKey returns e.Key if it is set.
// Otherwise, if e.ExtKey denotes a numpad key,
// EffectiveKey returns the equivalent e.Key value
// ('0'..'9', '.', '\n', '+', '-', '*', or '/').
// Otherwise, EffectiveKey returns zero.
func (e KeyEvent) EffectiveKey() byte {
	if e.Key != 0 {
		return e.Key
	}
	k := e.ExtKey
	switch {
	case k >= N0 && k <= N9:
		return byte(k - N0) + '0'
	case k == NDot:
		return '.'
	case k == NEnter:
		return '\n'
	case k == NAdd:
		return '+'
	case k == NSubtract:
		return '-'
	case k == NMultiply:
		return '*'
	case k == NDivide:
		return '/'
	}
	return 0
}

// Modifiers indicates modifier keys being held during an event.
// There is no way to differentiate between left and right modifier keys.
// As such, what KeyEvents get sent if the user does something unusual with both of a certain modifier key at once is (presently; TODO) undefined.
type Modifiers uintptr
const (
	Ctrl Modifiers = 1 << iota		// the canonical Ctrl keys ([TODO] on Mac OS X, Control on others)
	Alt						// the canonical Alt keys ([TODO] on Mac OS X, Meta on Unix systems, Alt on others)
	Shift						// the Shift keys
	// TODO add Super
)

func checkAreaSize(width int, height int, which string) {
	if width <= 0 || height <= 0 {
		panic(fmt.Errorf("invalid size %dx%d in %s", width, height, which))
	}
}

// NewArea creates a new Area with the given size and handler.
// It panics if handler is nil or if width or height is zero or negative.
func NewArea(width int, height int, handler AreaHandler) *Area {
	checkAreaSize(width, height, "NewArea()")
	if handler == nil {
		panic("handler passed to NewArea() must not be nil")
	}
	return &Area{
		sysData:		mksysdata(c_area),
		handler:		handler,
		initwidth:		width,
		initheight:		height,
	}
}

// SetSize sets the Area's internal drawing size.
// It has no effect on the actual control size.
// It panics if width or height is zero or negative.
func (a *Area) SetSize(width int, height int) {
	a.lock.Lock()
	defer a.lock.Unlock()

	checkAreaSize(width, height, "Area.SetSize()")
	if a.created {
		a.sysData.setAreaSize(width, height)
		return
	}
	a.initwidth = width
	a.initheight = height
}

func (a *Area) make(window *sysData) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.sysData.handler = a.handler
	err := a.sysData.make(window)
	if err != nil {
		return err
	}
	a.sysData.setAreaSize(a.initwidth, a.initheight)
	a.created = true
	return nil
}

func (a *Area) setRect(x int, y int, width int, height int, rr *[]resizerequest) {
	*rr = append(*rr, resizerequest{
		sysData:	a.sysData,
		x:		x,
		y:		y,
		width:	width,
		height:	height,
	})
}

func (a *Area) preferredSize() (width int, height int) {
	return a.sysData.preferredSize()
}

// internal function, but shared by all system implementations: &img.Pix[0] is not necessarily the first pixel in the image
func pixelDataPos(img *image.RGBA) int {
	return img.PixOffset(img.Rect.Min.X, img.Rect.Min.Y)
}

func pixelData(img *image.RGBA) *uint8 {
	return &img.Pix[pixelDataPos(img)]
}
