package render

import "image"

// ZoneLine is one world clock row: its label, part of the static layer, and
// its HH:MM, redrawn every tick.
type ZoneLine struct {
	Label string
	Time  string
}

// Text is everything the widget shows for one instant. It mirrors the clock
// model's snapshot field for field, but is declared here because render may
// import only the standard library, x/image and the font (specification
// Part 2); the shell copies one into the other.
type Text struct {
	Time     string
	DateLine string
	Zones    []ZoneLine
}

// Frame is one rendered picture: premultiplied BGRA, top down, rows packed
// with no padding. Pixels belongs to the Renderer and is overwritten by the
// next Render, which is what lets the steady state allocate nothing.
type Frame struct {
	Width  int
	Height int
	Pixels []byte

	// PanelBounds is the visible panel, outline included, within the frame.
	// The rest is the glow's transparent margin, which may hang off screen.
	PanelBounds image.Rectangle
}
