// Package product holds the product's name, decided once (WD2), so that every
// string derived from it (the settings folder, the exe, the Run value, the
// mutex and the tray tooltip) is built from one constant and a rename is one
// change plus a migration, never a search.
package product

// Name is baked into the settings path and the Run value, so changing it after
// release strands every existing user's settings: datum 0.11 applies.
const Name = "Thyme"

// Version is stamped into the start up log and, from WE8, the release. It is
// provisional until the author tags a release.
const Version = "0.1.0-dev"
