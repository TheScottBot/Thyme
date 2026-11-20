//go:build darwin && cgo

package desktop

/*
#include <stdlib.h>
#include "appkit_darwin.h"
*/
import "C"

import (
	"errors"
	"os"
	"path/filepath"
	"unsafe"
)

// ExeIsTemporary decides, once at start, whether Open at Login could point at
// something lasting. SMAppService registers the app bundle, so a program run
// bare, as go run does, has nothing to register; and a bundle in the temporary
// folder would soon be deleted.
func (platform *macPlatform) ExeIsTemporary() bool {
	exePath, exePathError := os.Executable()
	isTemporary := exePathError != nil || C.thymeIsInBundle() == 0
	if !isTemporary {
		isTemporary = IsInsideFolder(resolvedPath(exePath), resolvedPath(os.TempDir()))
	}
	if isTemporary {
		platform.shell.logger.Warn("not running from an app bundle of its own: Open at Login can be turned off but not on; build Thyme.app as README.md describes")
	}
	return isTemporary
}

// resolvedPath follows links, because the temporary folder is reached through
// /var, itself a link to /private/var. A path that cannot be resolved is
// compared as given.
func resolvedPath(linkedPath string) string {
	if resolved, resolveError := filepath.EvalSymlinks(linkedPath); resolveError == nil {
		return resolved
	}
	return linkedPath
}

// StartupEnabled asks the system, so the tick shows what will happen at the
// next login, including a user turning Thyme off in System Settings. A login
// item still waiting for approval will not run, so it reads as off.
func (platform *macPlatform) StartupEnabled() (bool, error) {
	return C.thymeLoginItemStatus() == C.thymeLoginItemEnabled, nil
}

func (platform *macPlatform) ChangeStartup(change StartupChange) error {
	if change.Origin == "" {
		return errors.New("a startup change must say which surface asked for it")
	}
	failure := C.thymeLoginItemSet(cBool(change.Enable))
	if failure == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(failure))
	return errors.New(C.GoString(failure))
}
