// Package render lays out and rasterises one frame of the widget into a
// premultiplied BGRA buffer. It is pure Go so the whole picture can be checked
// against golden images on any host; the shell only copies the bytes it
// returns to the screen.
package render
