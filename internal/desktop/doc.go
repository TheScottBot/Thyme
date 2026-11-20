// Package desktop is the shell: the window, the message loop, the timer, the
// menu, the tray icon, start at login and desktop ownership. core.go decides
// and is tested on every host; each platform's files only carry that out. It
// is the only package allowed to import C, syscall, unsafe or golang.org/x/sys,
// and it decides nothing with arithmetic in it; it prints what the pure
// packages return.
package desktop
