package desktop

// This file is the only one in Thyme that uses unsafe (specification WE5).
// Every function, constant and struct here was read from the Windows SDK
// 10.0.26100.0 headers (WinUser.h, wingdi.h, windef.h); none of them is
// exposed by golang.org/x/sys/windows v0.49.0, which is why each export is
// loaded from its DLL. Every unsafe use carries a comment naming the Win32
// contract that makes it valid.

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	gdi32  = windows.NewLazySystemDLL("gdi32.dll")

	procCreateWindowExW               = user32.NewProc("CreateWindowExW")
	procDefWindowProcW                = user32.NewProc("DefWindowProcW")
	procDispatchMessageW              = user32.NewProc("DispatchMessageW")
	procEnumDisplayMonitors           = user32.NewProc("EnumDisplayMonitors")
	procGetDC                         = user32.NewProc("GetDC")
	procGetDpiForWindow               = user32.NewProc("GetDpiForWindow")
	procGetMessageW                   = user32.NewProc("GetMessageW")
	procGetMonitorInfoW               = user32.NewProc("GetMonitorInfoW")
	procGetWindowRect                 = user32.NewProc("GetWindowRect")
	procLoadCursorW                   = user32.NewProc("LoadCursorW")
	procPostMessageW                  = user32.NewProc("PostMessageW")
	procPostQuitMessage               = user32.NewProc("PostQuitMessage")
	procRegisterClassExW              = user32.NewProc("RegisterClassExW")
	procReleaseDC                     = user32.NewProc("ReleaseDC")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procSetWindowPos                  = user32.NewProc("SetWindowPos")
	procShowWindow                    = user32.NewProc("ShowWindow")
	procTranslateMessage              = user32.NewProc("TranslateMessage")
	procUpdateLayeredWindow           = user32.NewProc("UpdateLayeredWindow")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procSelectObject       = gdi32.NewProc("SelectObject")
)

const (
	wsExLayered    = 0x00080000
	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000
	wsPopup        = 0x80000000

	wmDestroy        = 0x0002
	wmSettingChange  = 0x001A
	wmTimeChange     = 0x001E
	wmDisplayChange  = 0x007E
	wmNcHitTest      = 0x0084
	wmPowerBroadcast = 0x0218
	wmExitSizeMove   = 0x0232
	wmDpiChanged     = 0x02E0
	wmApp            = 0x8000

	htCaption = 2

	spiSetWorkArea = 0x002F

	pbtApmResumeAutomatic = 0x0012
	pbtApmResumeSuspend   = 0x0007

	swpNoSize     = 0x0001
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010

	swShowNoActivate = 4

	ulwAlpha   = 0x00000002
	acSrcOver  = 0x00
	acSrcAlpha = 0x01

	biRgb        = 0
	dibRgbColors = 0

	idcArrow = 32512

	// dpiAwarenessContextPerMonitorAwareV2 is ((DPI_AWARENESS_CONTEXT)-4).
	dpiAwarenessContextPerMonitorAwareV2 = ^uintptr(3)
)

type win32Point struct {
	X int32
	Y int32
}

type win32Size struct {
	Width  int32
	Height int32
}

type win32Rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type win32Message struct {
	Window  uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Point   win32Point
}

type win32WindowClass struct {
	Size            uint32
	Style           uint32
	WindowProcedure uintptr
	ClassExtra      int32
	WindowExtra     int32
	Instance        uintptr
	Icon            uintptr
	Cursor          uintptr
	Background      uintptr
	MenuName        *uint16
	ClassName       *uint16
	IconSmall       uintptr
}

type win32MonitorInfo struct {
	Size    uint32
	Monitor win32Rect
	Work    win32Rect
	Flags   uint32
}

type win32BitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ColoursUsed   uint32
	ColoursNeeded uint32
}

type win32BitmapInfo struct {
	Header  win32BitmapInfoHeader
	Colours [1]uint32
}

type win32BlendFunction struct {
	BlendOperation      byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

func setProcessPerMonitorAwareV2() error {
	if succeeded, _, callError := procSetProcessDpiAwarenessContext.Call(dpiAwarenessContextPerMonitorAwareV2); succeeded == 0 {
		return callError
	}
	return nil
}

func registerWindowClass(className *uint16, windowProcedure uintptr) error {
	arrowCursor, _, _ := procLoadCursorW.Call(0, idcArrow)
	windowClass := win32WindowClass{WindowProcedure: windowProcedure, ClassName: className, Cursor: arrowCursor}
	windowClass.Size = uint32(unsafe.Sizeof(windowClass))
	// RegisterClassExW reads the WNDCLASSEXW during the call only; the class
	// name pointer is kept by Go for the life of the process.
	if atom, _, callError := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&windowClass))); atom == 0 {
		return callError
	}
	return nil
}

func createLayeredToolWindow(className *uint16, windowTitle *uint16, left int32, top int32) (uintptr, error) {
	// CreateWindowExW copies both strings during the call.
	window, _, callError := procCreateWindowExW.Call(
		wsExLayered|wsExToolWindow|wsExNoActivate,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		wsPopup,
		uintptr(left), uintptr(top), 1, 1,
		0, 0, 0, 0,
	)
	if window == 0 {
		return 0, callError
	}
	return window, nil
}

func defaultWindowProcedure(window uintptr, message uint32, wParam uintptr, lParam uintptr) uintptr {
	result, _, _ := procDefWindowProcW.Call(window, uintptr(message), wParam, lParam)
	return result
}

// pumpMessages runs the message loop until WM_QUIT. GetMessageW returns -1 on
// an error, which ends the loop rather than spinning.
func pumpMessages() error {
	var message win32Message
	for {
		// GetMessageW writes one MSG into the struct, which lives on this
		// goroutine's stack for the whole call.
		result, _, callError := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		switch int32(result) {
		case 0:
			return nil
		case -1:
			return callError
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func postMessage(window uintptr, message uint32) error {
	if succeeded, _, callError := procPostMessageW.Call(window, uintptr(message), 0, 0); succeeded == 0 {
		return callError
	}
	return nil
}

func postQuitMessage() {
	procPostQuitMessage.Call(0)
}

func showWithoutActivating(window uintptr) {
	procShowWindow.Call(window, swShowNoActivate)
}

func dpiForWindow(window uintptr) int {
	dpi, _, _ := procGetDpiForWindow.Call(window)
	return int(dpi)
}

func windowRectangle(window uintptr) (win32Rect, error) {
	var rectangle win32Rect
	// GetWindowRect writes one RECT into the struct during the call.
	if succeeded, _, callError := procGetWindowRect.Call(window, uintptr(unsafe.Pointer(&rectangle))); succeeded == 0 {
		return win32Rect{}, callError
	}
	return rectangle, nil
}

func moveWindow(window uintptr, left int32, top int32) error {
	if succeeded, _, callError := procSetWindowPos.Call(window, 0, uintptr(left), uintptr(top), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate); succeeded == 0 {
		return callError
	}
	return nil
}

// rectangleFromLParam reads the RECT a WM_DPICHANGED message carries. The
// message's documented contract is that lParam points to a RECT valid for the
// duration of the handler, which is the only place this is called. The
// conversion goes through unsafe.Add on a nil pointer because the address
// arrives as an integer from Windows, not from Go.
func rectangleFromLParam(lParam uintptr) win32Rect {
	return *(*win32Rect)(unsafe.Add(unsafe.Pointer(nil), lParam))
}

var monitorWorkAreasCollected []win32Rect

// monitorEnumerationCallback is created once, because callbacks made with
// windows.NewCallback are never freed and there is a fixed number of them.
var monitorEnumerationCallback = windows.NewCallback(func(monitor uintptr, deviceContext uintptr, monitorRect uintptr, userData uintptr) uintptr {
	monitorInformation := win32MonitorInfo{}
	monitorInformation.Size = uint32(unsafe.Sizeof(monitorInformation))
	// GetMonitorInfoW fills the MONITORINFO during the call.
	if succeeded, _, _ := procGetMonitorInfoW.Call(monitor, uintptr(unsafe.Pointer(&monitorInformation))); succeeded != 0 {
		monitorWorkAreasCollected = append(monitorWorkAreasCollected, monitorInformation.Work)
	}
	return 1
})

// monitorWorkAreas lists every monitor's work area. EnumDisplayMonitors calls
// the callback synchronously on this thread before returning.
func monitorWorkAreas() ([]win32Rect, error) {
	monitorWorkAreasCollected = monitorWorkAreasCollected[:0]
	if succeeded, _, callError := procEnumDisplayMonitors.Call(0, 0, monitorEnumerationCallback, 0); succeeded == 0 {
		return nil, callError
	}
	return append([]win32Rect(nil), monitorWorkAreasCollected...), nil
}

// layeredBitmap is a top down 32 bit DIB selected into a memory device
// context, sized once per frame size and reused every tick.
type layeredBitmap struct {
	memoryContext  uintptr
	bitmap         uintptr
	previousObject uintptr
	pixels         []byte
	width          int
	height         int
}

func newLayeredBitmap(width int, height int) (*layeredBitmap, error) {
	memoryContext, _, callError := procCreateCompatibleDC.Call(0)
	if memoryContext == 0 {
		return nil, callError
	}
	bitmapInformation := win32BitmapInfo{Header: win32BitmapInfoHeader{
		Width: int32(width), Height: -int32(height), Planes: 1, BitCount: 32, Compression: biRgb,
	}}
	bitmapInformation.Header.Size = uint32(unsafe.Sizeof(bitmapInformation.Header))
	var bitsAddress uintptr
	// CreateDIBSection reads the BITMAPINFO during the call and writes the
	// address of the bitmap's pixel memory into bitsAddress.
	bitmap, _, callError := procCreateDIBSection.Call(memoryContext, uintptr(unsafe.Pointer(&bitmapInformation)), dibRgbColors, uintptr(unsafe.Pointer(&bitsAddress)), 0, 0)
	if bitmap == 0 || bitsAddress == 0 {
		procDeleteDC.Call(memoryContext)
		return nil, callError
	}
	previousObject, _, _ := procSelectObject.Call(memoryContext, bitmap)
	return &layeredBitmap{
		memoryContext:  memoryContext,
		bitmap:         bitmap,
		previousObject: previousObject,
		// The DIB section owns width times height times four bytes at
		// bitsAddress until DeleteObject, which release calls only after this
		// slice is dropped. A negative height makes the rows top down, the
		// order the renderer writes.
		pixels: unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(nil), bitsAddress)), width*height*4),
		width:  width,
		height: height,
	}, nil
}

func (bitmap *layeredBitmap) release() {
	bitmap.pixels = nil
	procSelectObject.Call(bitmap.memoryContext, bitmap.previousObject)
	procDeleteObject.Call(bitmap.bitmap)
	procDeleteDC.Call(bitmap.memoryContext)
}

// updateLayeredWindow hands the bitmap to the window with per pixel alpha.
// moveTo nil leaves the window where it is, which is what keeps a tick from
// fighting a drag in progress.
func updateLayeredWindow(window uintptr, bitmap *layeredBitmap, moveTo *win32Point) error {
	screenContext, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screenContext)
	size := win32Size{Width: int32(bitmap.width), Height: int32(bitmap.height)}
	sourceOrigin := win32Point{}
	blend := win32BlendFunction{BlendOperation: acSrcOver, SourceConstantAlpha: 255, AlphaFormat: acSrcAlpha}
	var destination uintptr
	if moveTo != nil {
		destination = uintptr(unsafe.Pointer(moveTo))
	}
	// UpdateLayeredWindow reads the point, size and blend function during the
	// call only.
	succeeded, _, callError := procUpdateLayeredWindow.Call(
		window, screenContext, destination,
		uintptr(unsafe.Pointer(&size)), bitmap.memoryContext, uintptr(unsafe.Pointer(&sourceOrigin)),
		0, uintptr(unsafe.Pointer(&blend)), ulwAlpha,
	)
	if succeeded == 0 {
		return callError
	}
	return nil
}

func utf16OrPanic(value string) *uint16 {
	pointer, conversionError := syscall.UTF16PtrFromString(value)
	if conversionError != nil {
		panic("a constant string holds a NUL: " + value)
	}
	return pointer
}

// WE6 bindings: the tray icon, the menu, desktop ownership and the window's
// text. Read from WinUser.h, wingdi.h and shellapi.h of SDK 10.0.26100.0.

var (
	shell32 = windows.NewLazySystemDLL("shell32.dll")

	procAppendMenuW            = user32.NewProc("AppendMenuW")
	procCreateIconIndirect     = user32.NewProc("CreateIconIndirect")
	procCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	procDestroyIcon            = user32.NewProc("DestroyIcon")
	procDestroyMenu            = user32.NewProc("DestroyMenu")
	procDestroyWindow          = user32.NewProc("DestroyWindow")
	procFindWindowExW          = user32.NewProc("FindWindowExW")
	procFindWindowW            = user32.NewProc("FindWindowW")
	procGetDpiForSystem        = user32.NewProc("GetDpiForSystem")
	procGetSystemMetricsForDpi = user32.NewProc("GetSystemMetricsForDpi")
	procRegisterWindowMessageW = user32.NewProc("RegisterWindowMessageW")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procSetWindowLongPtrW      = user32.NewProc("SetWindowLongPtrW")
	procSetWindowTextW         = user32.NewProc("SetWindowTextW")
	procTrackPopupMenu         = user32.NewProc("TrackPopupMenu")

	procCreateBitmap = gdi32.NewProc("CreateBitmap")

	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
)

const (
	wmNull        = 0x0000
	wmContextMenu = 0x007B

	ninSelect    = wmUser + 0
	ninfKey      = 0x1
	ninKeySelect = ninSelect | ninfKey
	wmUser       = 0x0400

	nimAdd        = 0x00000000
	nimDelete     = 0x00000002
	nimSetVersion = 0x00000004

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004
	nifShowTip = 0x00000080

	notifyIconVersion4 = 4

	mfString    = 0x00000000
	mfGrayed    = 0x00000001
	mfChecked   = 0x00000008
	mfPopup     = 0x00000010
	mfSeparator = 0x00000800

	tpmRightButton = 0x0002
	tpmNoNotify    = 0x0080
	tpmReturnCmd   = 0x0100

	swpNoMove = 0x0002

	smCxSmallIcon = 49

	// gwlpHwndParent is (-8). On a top level window it sets the owner, which
	// is how the widget is owned by the desktop's view.
	gwlpHwndParent = ^uintptr(7)

	// hwndTopmost is ((HWND)-1); hwndBottom is ((HWND)1).
	hwndTopmost = ^uintptr(0)
	hwndBottom  = uintptr(1)
)

type win32NotifyIconData struct {
	Size             uint32
	Window           uintptr
	Identifier       uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	Item             windows.GUID
	BalloonIcon      uintptr
}

type win32IconInfo struct {
	IsIcon   int32
	HotspotX uint32
	HotspotY uint32
	Mask     uintptr
	Colour   uintptr
}

func destroyWindow(window uintptr) {
	procDestroyWindow.Call(window)
}

func registerWindowMessage(name string) uint32 {
	// RegisterWindowMessageW reads the string during the call.
	message, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(utf16OrPanic(name))))
	return uint32(message)
}

func setWindowText(window uintptr, text string) error {
	textPointer, conversionError := syscall.UTF16PtrFromString(text)
	if conversionError != nil {
		return conversionError
	}
	// SetWindowTextW copies the string during the call.
	if succeeded, _, callError := procSetWindowTextW.Call(window, uintptr(unsafe.Pointer(textPointer))); succeeded == 0 {
		return callError
	}
	return nil
}

func findWindowByClass(parent uintptr, className string) uintptr {
	// FindWindowW and FindWindowExW read the class name during the call.
	classPointer := uintptr(unsafe.Pointer(utf16OrPanic(className)))
	if parent == 0 {
		window, _, _ := procFindWindowW.Call(classPointer, 0)
		return window
	}
	window, _, _ := procFindWindowExW.Call(parent, 0, classPointer, 0)
	return window
}

func setOwner(window uintptr, owner uintptr) {
	procSetWindowLongPtrW.Call(window, gwlpHwndParent, owner)
}

func setZOrder(window uintptr, insertAfter uintptr) error {
	if succeeded, _, callError := procSetWindowPos.Call(window, insertAfter, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate); succeeded == 0 {
		return callError
	}
	return nil
}

func setForegroundWindow(window uintptr) {
	procSetForegroundWindow.Call(window)
}

func smallIconSize() int {
	systemDpi, _, _ := procGetDpiForSystem.Call()
	size, _, _ := procGetSystemMetricsForDpi.Call(smCxSmallIcon, systemDpi)
	return int(size)
}

// createIcon builds an icon from premultiplied BGRA pixels: a 32 bit top down
// colour bitmap whose alpha is the shape, and an all zero mask, which an
// alpha icon still requires.
func createIcon(pixels []byte, width int, height int) (uintptr, error) {
	bitmapInformation := win32BitmapInfo{Header: win32BitmapInfoHeader{
		Width: int32(width), Height: -int32(height), Planes: 1, BitCount: 32, Compression: biRgb,
	}}
	bitmapInformation.Header.Size = uint32(unsafe.Sizeof(bitmapInformation.Header))
	var bitsAddress uintptr
	// As in newLayeredBitmap: the BITMAPINFO is read during the call and the
	// pixel memory's address is written to bitsAddress.
	colourBitmap, _, callError := procCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bitmapInformation)), dibRgbColors, uintptr(unsafe.Pointer(&bitsAddress)), 0, 0)
	if colourBitmap == 0 || bitsAddress == 0 {
		return 0, callError
	}
	defer procDeleteObject.Call(colourBitmap)
	// The DIB section owns width times height times four bytes at bitsAddress
	// until the deferred DeleteObject above.
	copy(unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(nil), bitsAddress)), width*height*4), pixels)

	// Monochrome rows are padded to 16 bits.
	maskBits := make([]byte, (width+15)/16*2*height)
	// CreateBitmap copies the mask bits during the call.
	maskBitmap, _, callError := procCreateBitmap.Call(uintptr(width), uintptr(height), 1, 1, uintptr(unsafe.Pointer(&maskBits[0])))
	if maskBitmap == 0 {
		return 0, callError
	}
	defer procDeleteObject.Call(maskBitmap)

	iconInformation := win32IconInfo{IsIcon: 1, Mask: maskBitmap, Colour: colourBitmap}
	// CreateIconIndirect copies both bitmaps, so they are deleted after it.
	icon, _, callError := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&iconInformation)))
	if icon == 0 {
		return 0, callError
	}
	return icon, nil
}

func destroyIcon(icon uintptr) {
	procDestroyIcon.Call(icon)
}

func notifyIcon(operation uintptr, notifyData *win32NotifyIconData) error {
	notifyData.Size = uint32(unsafe.Sizeof(*notifyData))
	// Shell_NotifyIconW reads the NOTIFYICONDATAW during the call only.
	if succeeded, _, callError := procShellNotifyIconW.Call(operation, uintptr(unsafe.Pointer(notifyData))); succeeded == 0 {
		return callError
	}
	return nil
}

func createPopupMenu() uintptr {
	menu, _, _ := procCreatePopupMenu.Call()
	return menu
}

func appendMenuItem(menu uintptr, flags uintptr, identifierOrSubmenu uintptr, label string) error {
	var labelAddress uintptr
	if label != "" {
		// AppendMenuW copies the label during the call.
		labelAddress = uintptr(unsafe.Pointer(utf16OrPanic(label)))
	}
	if succeeded, _, callError := procAppendMenuW.Call(menu, flags, identifierOrSubmenu, labelAddress); succeeded == 0 {
		return callError
	}
	return nil
}

// trackPopupMenu shows the menu and returns the chosen command, or zero when
// it is dismissed. It runs a modal loop; posted ticks are still dispatched
// inside it (WE5 experiment), so the clock keeps moving.
func trackPopupMenu(menu uintptr, window uintptr, x int32, y int32) int {
	command, _, _ := procTrackPopupMenu.Call(menu, tpmReturnCmd|tpmRightButton|tpmNoNotify, uintptr(x), uintptr(y), 0, window, 0)
	return int(command)
}

func destroyMenu(menu uintptr) {
	procDestroyMenu.Call(menu)
}

// signedLowWord and signedHighWord split a packed pair of screen coordinates,
// which are signed because a monitor left of or above the primary one has
// negative coordinates.
func signedLowWord(packed uintptr) int32  { return int32(int16(uint16(packed))) }
func signedHighWord(packed uintptr) int32 { return int32(int16(uint16(packed >> 16))) }
