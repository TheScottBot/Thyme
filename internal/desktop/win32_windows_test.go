package desktop

import (
	"testing"
	"unsafe"
)

// Each struct is passed to Win32 by pointer, so its Go layout must match the
// SDK header's byte for byte. Sizes are for 64 bit Windows, the only kind WD10
// targets, worked out from the 10.0.26100.0 headers' field lists (recorded in
// docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md, WE5 Plan).
func TestWin32Structs_MatchTheSdkSizes(t *testing.T) {
	testCases := []struct {
		name         string
		actualSize   uintptr
		expectedSize uintptr
	}{
		{name: "MSG", actualSize: unsafe.Sizeof(win32Message{}), expectedSize: 48},
		{name: "WNDCLASSEXW", actualSize: unsafe.Sizeof(win32WindowClass{}), expectedSize: 80},
		{name: "MONITORINFO", actualSize: unsafe.Sizeof(win32MonitorInfo{}), expectedSize: 40},
		{name: "BITMAPINFOHEADER", actualSize: unsafe.Sizeof(win32BitmapInfoHeader{}), expectedSize: 40},
		{name: "BITMAPINFO", actualSize: unsafe.Sizeof(win32BitmapInfo{}), expectedSize: 44},
		{name: "BLENDFUNCTION", actualSize: unsafe.Sizeof(win32BlendFunction{}), expectedSize: 4},
		{name: "POINT", actualSize: unsafe.Sizeof(win32Point{}), expectedSize: 8},
		{name: "SIZE", actualSize: unsafe.Sizeof(win32Size{}), expectedSize: 8},
		{name: "RECT", actualSize: unsafe.Sizeof(win32Rect{}), expectedSize: 16},
	}
	for _, testCase := range testCases {
		if testCase.actualSize != testCase.expectedSize {
			t.Errorf("%s is %d bytes, the SDK's is %d", testCase.name, testCase.actualSize, testCase.expectedSize)
		}
	}
}
