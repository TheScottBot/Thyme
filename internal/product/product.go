// Package product holds the product's name, decided once (WD2), so that every
// string derived from it (the settings folder, the exe, the Run value, the
// mutex and the tray tooltip) is built from one constant and a rename is one
// change plus a migration, never a search.
package product

// Name is baked into the settings path and the Run value, so changing it after
// release strands every existing user's settings: datum 0.11 applies.
const Name = "Thyme"
