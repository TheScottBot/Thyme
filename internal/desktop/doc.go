// Package desktop is the Windows shell: the window, the message loop, the
// timer, the menu, the tray icon, the registry and desktop ownership. It is the
// only package allowed to import syscall, unsafe or golang.org/x/sys, and it
// decides nothing with arithmetic in it; it prints what the pure packages
// return.
package desktop
