// The C face of appkit_darwin.m, shared with cgo. Every call here runs on the
// main thread, except thymePostTick. Positions are points, measured from the
// top left of the primary screen, which is what the shell works in.
#ifndef THYME_APPKIT_DARWIN_H
#define THYME_APPKIT_DARWIN_H

#include <stdint.h>

typedef struct {
    int left;
    int top;
    int right;
    int bottom;
} ThymeRect;

void thymeInitApplication(void);
void thymeRunApplication(void);
void thymeStopApplication(void);

// thymeCreateWindow returns the backing scale of the screen it landed on.
double thymeCreateWindow(int x, int y);
void thymeShowWindow(void);
int thymePresent(const uint8_t *pixels, int width, int height, double scale, int hasMove, int moveX, int moveY);
ThymeRect thymeWindowBounds(void);
int thymeWorkAreas(ThymeRect *areas, int maximum);
void thymeMoveWindow(int x, int y);
void thymeApplyLevel(int alwaysOnTop);
void thymeSetAccessibleText(const char *text);

int thymeAddStatusItem(const uint8_t *pixels, int width, int height, double points);
void thymeRemoveStatusItem(void);

void *thymeMenuCreate(const char *title);
void thymeMenuAddItem(void *menu, const char *label, int tag, int checked, int enabled);
void thymeMenuAddSeparator(void *menu);
void thymeMenuAddSubmenu(void *menu, void *submenu, const char *label);
int thymeMenuShow(void *menu, int x, int y);

void thymePostTick(void);

// The current system time zone's name, to be freed by the caller.
char *thymeSystemTimeZoneName(void);

int thymeIsInBundle(void);

// The values of SMAppServiceStatus.
enum {
    thymeLoginItemNotRegistered = 0,
    thymeLoginItemEnabled = 1,
    thymeLoginItemRequiresApproval = 2,
    thymeLoginItemNotFound = 3
};
int thymeLoginItemStatus(void);
// Returns NULL on success, or a message to be freed by the caller.
char *thymeLoginItemSet(int enable);

// Implemented in Go, called on the main thread.
extern void thymeGoTick(void);
extern void thymeGoContextMenu(int x, int y);
extern void thymeGoStatusItemClicked(int x, int y);
extern void thymeGoMoveFinished(void);
extern void thymeGoScreensChanged(void);
extern void thymeGoBackingChanged(double scale);
extern void thymeGoClockChanged(void);
extern void thymeGoResumed(void);

#endif
