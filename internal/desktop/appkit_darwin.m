//go:build darwin && cgo

// The macOS window, menu, status item and login item. It decides nothing: it
// carries out what the shell in core.go asks and reports what AppKit tells it.

#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>
#import <ServiceManagement/ServiceManagement.h>

#include <math.h>
#include <string.h>

#include "appkit_darwin.h"

@interface ThymePanel : NSPanel
@end

@interface ThymeView : NSView
@end

// ThymeTarget receives menu choices, status item clicks, notifications and
// window delegate calls, for as long as the process lives.
@interface ThymeTarget : NSObject <NSWindowDelegate>
@end

static ThymePanel *widgetWindow;
static ThymeView *widgetView;
static NSStatusItem *statusItem;
static ThymeTarget *target;
static id appNapActivity;
static int chosenCommand;

// primaryHeight is the height of the screen AppKit's coordinates start from,
// which turns AppKit's rising Y into the shell's falling one and back.
static CGFloat primaryHeight(void) {
    NSScreen *primary = NSScreen.screens.firstObject;
    return primary == nil ? 0 : primary.frame.size.height;
}

static ThymeRect shellRect(NSRect rectangle) {
    CGFloat height = primaryHeight();
    ThymeRect converted = {
        .left = (int)floor(NSMinX(rectangle)),
        .top = (int)floor(height - NSMaxY(rectangle)),
        .right = (int)ceil(NSMaxX(rectangle)),
        .bottom = (int)ceil(height - NSMinY(rectangle)),
    };
    return converted;
}

// createImage copies the frame, which is only valid during the call. The bytes
// are premultiplied BGRA, as the renderer produces.
static CGImageRef createImage(const uint8_t *pixels, int width, int height) {
    CFDataRef data = CFDataCreate(kCFAllocatorDefault, pixels, (CFIndex)width * height * 4);
    if (data == NULL) {
        return NULL;
    }
    CGDataProviderRef provider = CGDataProviderCreateWithCFData(data);
    CFRelease(data);
    if (provider == NULL) {
        return NULL;
    }
    CGColorSpaceRef colourSpace = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
    CGBitmapInfo bitmapInfo = kCGBitmapByteOrder32Little | (CGBitmapInfo)kCGImageAlphaPremultipliedFirst;
    CGImageRef image = CGImageCreate(width, height, 8, 32, (size_t)width * 4, colourSpace, bitmapInfo, provider, NULL, false, kCGRenderingIntentDefault);
    CGColorSpaceRelease(colourSpace);
    CGDataProviderRelease(provider);
    return image;
}

static void reportContextMenu(void) {
    NSPoint mouse = NSEvent.mouseLocation;
    thymeGoContextMenu((int)mouse.x, (int)(primaryHeight() - mouse.y));
}

@implementation ThymePanel
// The widget never takes focus from what the user is doing.
- (BOOL)canBecomeKeyWindow {
    return NO;
}

- (BOOL)canBecomeMainWindow {
    return NO;
}

// The shell has already decided where the widget goes, and the glow's
// transparent margin may hang off the screen, which AppKit would otherwise
// correct by moving the window.
- (NSRect)constrainFrameRect:(NSRect)frameRect toScreen:(NSScreen *)screen {
    return frameRect;
}
@end

@implementation ThymeView
- (BOOL)acceptsFirstMouse:(NSEvent *)event {
    return YES;
}

- (BOOL)isAccessibilityElement {
    return YES;
}

- (NSAccessibilityRole)accessibilityRole {
    return NSAccessibilityStaticTextRole;
}

- (void)rightMouseDown:(NSEvent *)event {
    reportContextMenu();
}

// The drag is tracked here rather than left to the window server, so its end
// is known exactly, and the shell clamps and saves once per drag. Ticks still
// arrive during it, because the main queue is served in tracking mode.
- (void)mouseDown:(NSEvent *)event {
    if ((event.modifierFlags & NSEventModifierFlagControl) != 0) {
        reportContextMenu();
        return;
    }
    NSWindow *window = self.window;
    NSPoint startMouse = NSEvent.mouseLocation;
    NSPoint startOrigin = window.frame.origin;
    BOOL moved = NO;
    while (YES) {
        NSEvent *next = [NSApp nextEventMatchingMask:NSEventMaskLeftMouseDragged | NSEventMaskLeftMouseUp
                                           untilDate:NSDate.distantFuture
                                              inMode:NSEventTrackingRunLoopMode
                                             dequeue:YES];
        if (next.type == NSEventTypeLeftMouseUp) {
            break;
        }
        NSPoint mouse = NSEvent.mouseLocation;
        [window setFrameOrigin:NSMakePoint(startOrigin.x + mouse.x - startMouse.x, startOrigin.y + mouse.y - startMouse.y)];
        moved = YES;
    }
    if (moved) {
        thymeGoMoveFinished();
    }
}
@end

@implementation ThymeTarget
- (void)menuItemChosen:(NSMenuItem *)item {
    chosenCommand = (int)item.tag;
}

- (void)statusItemClicked:(id)sender {
    NSStatusBarButton *button = statusItem.button;
    NSRect inWindow = [button convertRect:button.bounds toView:nil];
    NSRect onScreen = [button.window convertRectToScreen:inWindow];
    thymeGoStatusItemClicked((int)NSMinX(onScreen), (int)(primaryHeight() - NSMinY(onScreen)));
}

// AppKit sends this from inside setFrame, in the middle of thymePresent. Going
// back to the shell there would redraw inside a redraw, and the outer one would
// then finish with a frame and insets for the old scale, so it waits its turn.
- (void)windowDidChangeBackingProperties:(NSNotification *)notification {
    dispatch_async(dispatch_get_main_queue(), ^{
        thymeGoBackingChanged(widgetWindow.backingScaleFactor);
    });
}

- (void)screensChanged:(NSNotification *)notification {
    thymeGoScreensChanged();
}

- (void)clockChanged:(NSNotification *)notification {
    thymeGoClockChanged();
}

- (void)resumed:(NSNotification *)notification {
    thymeGoResumed();
}
@end

void thymeInitApplication(void) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        // No Dock icon and no menu bar, even when run outside the bundle.
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
        target = [[ThymeTarget alloc] init];

        NSNotificationCenter *centre = NSNotificationCenter.defaultCenter;
        [centre addObserver:target selector:@selector(screensChanged:) name:NSApplicationDidChangeScreenParametersNotification object:nil];
        [centre addObserver:target selector:@selector(clockChanged:) name:NSSystemClockDidChangeNotification object:nil];
        [centre addObserver:target selector:@selector(clockChanged:) name:NSSystemTimeZoneDidChangeNotification object:nil];
        [NSWorkspace.sharedWorkspace.notificationCenter addObserver:target selector:@selector(resumed:) name:NSWorkspaceDidWakeNotification object:nil];

        // A clock showing seconds must keep time while the user is elsewhere,
        // which App Nap would otherwise stop it doing. Idle sleep is unaffected.
        appNapActivity = [NSProcessInfo.processInfo beginActivityWithOptions:NSActivityUserInitiatedAllowingIdleSystemSleep reason:@"Showing the time"];
    }
}

void thymeRunApplication(void) {
    [NSApp run];
}

// stop only takes effect after the next event, so one is posted.
void thymeStopApplication(void) {
    @autoreleasepool {
        [NSApp stop:nil];
        NSEvent *wake = [NSEvent otherEventWithType:NSEventTypeApplicationDefined
                                           location:NSZeroPoint
                                      modifierFlags:0
                                          timestamp:0
                                       windowNumber:0
                                            context:nil
                                            subtype:0
                                              data1:0
                                              data2:0];
        [NSApp postEvent:wake atStart:YES];
    }
}

double thymeCreateWindow(int x, int y) {
    @autoreleasepool {
        NSRect frame = NSMakeRect(x, primaryHeight() - y - 1, 1, 1);
        widgetWindow = [[ThymePanel alloc] initWithContentRect:frame
                                                     styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
                                                       backing:NSBackingStoreBuffered
                                                         defer:NO];
        widgetWindow.releasedWhenClosed = NO;
        widgetWindow.opaque = NO;
        widgetWindow.backgroundColor = NSColor.clearColor;
        widgetWindow.hasShadow = NO;
        widgetWindow.hidesOnDeactivate = NO;
        // Every Space, and untouched by Show Desktop and Mission Control.
        widgetWindow.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces | NSWindowCollectionBehaviorStationary | NSWindowCollectionBehaviorIgnoresCycle;
        widgetWindow.delegate = target;

        widgetView = [[ThymeView alloc] initWithFrame:NSMakeRect(0, 0, 1, 1)];
        widgetView.wantsLayer = YES;
        widgetView.layerContentsRedrawPolicy = NSViewLayerContentsRedrawNever;
        widgetView.layer.contentsGravity = kCAGravityResize;
        widgetWindow.contentView = widgetView;
        return widgetWindow.backingScaleFactor;
    }
}

void thymeShowWindow(void) {
    [widgetWindow orderFrontRegardless];
}

int thymePresent(const uint8_t *pixels, int width, int height, double scale, int hasMove, int moveX, int moveY) {
    @autoreleasepool {
        CGImageRef image = createImage(pixels, width, height);
        if (image == NULL) {
            return -1;
        }
        // The top left corner stays put unless a move was asked for, so a size
        // change grows the widget down and to the right.
        NSRect current = widgetWindow.frame;
        CGFloat left = hasMove ? moveX : NSMinX(current);
        CGFloat top = hasMove ? primaryHeight() - moveY : NSMaxY(current);
        NSSize size = NSMakeSize(width / scale, height / scale);
        NSRect next = NSMakeRect(left, top - size.height, size.width, size.height);

        [CATransaction begin];
        [CATransaction setDisableActions:YES];
        if (!NSEqualRects(next, current)) {
            [widgetWindow setFrame:next display:NO];
        }
        widgetView.layer.contentsScale = scale;
        widgetView.layer.contents = (__bridge id)image;
        [CATransaction commit];
        CGImageRelease(image);
        return 0;
    }
}

ThymeRect thymeWindowBounds(void) {
    return shellRect(widgetWindow.frame);
}

// visibleFrame leaves out the menu bar and the Dock.
int thymeWorkAreas(ThymeRect *areas, int maximum) {
    @autoreleasepool {
        int count = 0;
        for (NSScreen *screen in NSScreen.screens) {
            if (count == maximum) {
                break;
            }
            areas[count] = shellRect(screen.visibleFrame);
            count++;
        }
        return count;
    }
}

void thymeMoveWindow(int x, int y) {
    [widgetWindow setFrameTopLeftPoint:NSMakePoint(x, primaryHeight() - y)];
}

// Off sits just above the desktop icons, below every ordinary window, which is
// what off means on Windows too.
void thymeApplyLevel(int alwaysOnTop) {
    widgetWindow.level = alwaysOnTop ? NSFloatingWindowLevel : CGWindowLevelForKey(kCGDesktopIconWindowLevelKey) + 1;
}

void thymeSetAccessibleText(const char *text) {
    @autoreleasepool {
        NSString *value = [NSString stringWithUTF8String:text];
        widgetView.accessibilityLabel = value;
        widgetWindow.title = value;
    }
}

int thymeAddStatusItem(const uint8_t *pixels, int width, int height, double points) {
    @autoreleasepool {
        if (statusItem != nil) {
            return 0;
        }
        CGImageRef image = createImage(pixels, width, height);
        if (image == NULL) {
            return -1;
        }
        statusItem = [NSStatusBar.systemStatusBar statusItemWithLength:NSSquareStatusItemLength];
        NSStatusBarButton *button = statusItem.button;
        button.image = [[NSImage alloc] initWithCGImage:image size:NSMakeSize(points, points)];
        button.accessibilityLabel = @"Thyme";
        button.toolTip = @"Thyme";
        button.target = target;
        button.action = @selector(statusItemClicked:);
        [button sendActionOn:NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp];
        CGImageRelease(image);
        return 0;
    }
}

void thymeRemoveStatusItem(void) {
    if (statusItem != nil) {
        [NSStatusBar.systemStatusBar removeStatusItem:statusItem];
        statusItem = nil;
    }
}

void *thymeMenuCreate(const char *title) {
    @autoreleasepool {
        NSMenu *menu = [[NSMenu alloc] initWithTitle:[NSString stringWithUTF8String:title]];
        menu.autoenablesItems = NO;
        return (void *)CFBridgingRetain(menu);
    }
}

void thymeMenuAddItem(void *menu, const char *label, int tag, int checked, int enabled) {
    @autoreleasepool {
        NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:[NSString stringWithUTF8String:label]
                                                      action:@selector(menuItemChosen:)
                                               keyEquivalent:@""];
        item.target = target;
        item.tag = tag;
        item.state = checked ? NSControlStateValueOn : NSControlStateValueOff;
        item.enabled = enabled != 0;
        [(__bridge NSMenu *)menu addItem:item];
    }
}

void thymeMenuAddSeparator(void *menu) {
    @autoreleasepool {
        [(__bridge NSMenu *)menu addItem:NSMenuItem.separatorItem];
    }
}

// The parent keeps the submenu, so this gives up the reference from its create.
void thymeMenuAddSubmenu(void *menu, void *submenu, const char *label) {
    @autoreleasepool {
        NSMenu *child = CFBridgingRelease(submenu);
        NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:[NSString stringWithUTF8String:label] action:NULL keyEquivalent:@""];
        item.submenu = child;
        [(__bridge NSMenu *)menu addItem:item];
    }
}

// thymeMenuShow blocks until the menu closes, and gives up the reference from
// its create. It returns the chosen tag, or zero if dismissed.
int thymeMenuShow(void *menu, int x, int y) {
    @autoreleasepool {
        NSMenu *shown = CFBridgingRelease(menu);
        chosenCommand = 0;
        [shown popUpMenuPositioningItem:nil atLocation:NSMakePoint(x, primaryHeight() - y) inView:nil];
        return chosenCommand;
    }
}

void thymePostTick(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        thymeGoTick();
    });
}

char *thymeSystemTimeZoneName(void) {
    @autoreleasepool {
        [NSTimeZone resetSystemTimeZone];
        return strdup(NSTimeZone.systemTimeZone.name.UTF8String);
    }
}

int thymeIsInBundle(void) {
    @autoreleasepool {
        return NSBundle.mainBundle.bundleIdentifier != nil ? 1 : 0;
    }
}

int thymeLoginItemStatus(void) {
    return (int)SMAppService.mainAppService.status;
}

// A login item the user must approve opens the page where they can, so the
// click is not left looking as if nothing happened.
char *thymeLoginItemSet(int enable) {
    @autoreleasepool {
        SMAppService *service = SMAppService.mainAppService;
        NSError *failure = nil;
        BOOL succeeded = enable ? [service registerAndReturnError:&failure] : [service unregisterAndReturnError:&failure];
        if (!succeeded) {
            return strdup(failure.localizedDescription.UTF8String);
        }
        if (enable && service.status == SMAppServiceStatusRequiresApproval) {
            [SMAppService openSystemSettingsLoginItems];
        }
        return NULL;
    }
}
