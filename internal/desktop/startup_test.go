package desktop

import (
	"errors"
	"testing"
)

// fakeStartupStore stands in for the two registry values, with a failure
// switch per operation so partial failures can be driven (datum 0.14).
type fakeStartupStore struct {
	runValue           string
	hasRunValue        bool
	approvalBytes      []byte
	hasApproval        bool
	failWriteRun       bool
	failDeleteRun      bool
	failDeleteApproval bool
	failReads          bool
}

var errFakeRegistry = errors.New("fake registry failure")

func (store *fakeStartupStore) ReadRunValue() (string, bool, error) {
	if store.failReads {
		return "", false, errFakeRegistry
	}
	return store.runValue, store.hasRunValue, nil
}

func (store *fakeStartupStore) WriteRunValue(value string) error {
	if store.failWriteRun {
		return errFakeRegistry
	}
	store.runValue, store.hasRunValue = value, true
	return nil
}

func (store *fakeStartupStore) DeleteRunValue() error {
	if store.failDeleteRun {
		return errFakeRegistry
	}
	store.runValue, store.hasRunValue = "", false
	return nil
}

func (store *fakeStartupStore) ReadApproval() ([]byte, bool, error) {
	if store.failReads {
		return nil, false, errFakeRegistry
	}
	return store.approvalBytes, store.hasApproval, nil
}

func (store *fakeStartupStore) DeleteApproval() error {
	if store.failDeleteApproval {
		return errFakeRegistry
	}
	store.approvalBytes, store.hasApproval = nil, false
	return nil
}

const exePathWithSpace = `C:\Program Files\Thyme\thyme.exe`

// Twelve bytes as Task Manager wrote them for two apps the author confirmed
// it shows as Disabled (WE6 Plan): first byte 0x01.
var taskManagerDisabled = []byte{0x01, 0, 0, 0, 0x4a, 0x7b, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66}

func TestRunValueFor_QuotesAPathContainingASpace(t *testing.T) {
	if value := RunValueFor(exePathWithSpace); value != `"C:\Program Files\Thyme\thyme.exe"` {
		t.Errorf("expected the path in double quotes, got %s", value)
	}
}

func TestStartupEnabled_ReadsBothValuesFresh(t *testing.T) {
	testCases := []struct {
		name     string
		store    fakeStartupStore
		expected bool
	}{
		{name: "no Run value", store: fakeStartupStore{}, expected: false},
		{name: "our Run value and no approval entry", store: fakeStartupStore{runValue: RunValueFor(exePathWithSpace), hasRunValue: true}, expected: true},
		{name: "our Run value, disabled in Task Manager", store: fakeStartupStore{runValue: RunValueFor(exePathWithSpace), hasRunValue: true, approvalBytes: taskManagerDisabled, hasApproval: true}, expected: false},
		{name: "our Run value, approval entry with an even first byte", store: fakeStartupStore{runValue: RunValueFor(exePathWithSpace), hasRunValue: true, approvalBytes: []byte{0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, hasApproval: true}, expected: true},
		{name: "a Run value for an exe somewhere else", store: fakeStartupStore{runValue: `"D:\Old\thyme.exe"`, hasRunValue: true}, expected: false},
		{name: "an empty approval entry", store: fakeStartupStore{runValue: RunValueFor(exePathWithSpace), hasRunValue: true, approvalBytes: []byte{}, hasApproval: true}, expected: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			enabled, readError := StartupEnabled(&testCase.store, exePathWithSpace)
			if readError != nil {
				t.Fatalf("unexpected error: %v", readError)
			}
			if enabled != testCase.expected {
				t.Errorf("expected %v, got %v", testCase.expected, enabled)
			}
		})
	}
}

func TestStartupEnabled_ReportsAReadFailure(t *testing.T) {
	if _, readError := StartupEnabled(&fakeStartupStore{failReads: true}, exePathWithSpace); readError == nil {
		t.Error("expected a failed read to be reported, not shown as off")
	}
}

func TestChangeStartup_EnablingWritesTheRunValueAndClearsADisable(t *testing.T) {
	store := &fakeStartupStore{approvalBytes: taskManagerDisabled, hasApproval: true}

	if changeError := ChangeStartup(store, exePathWithSpace, StartupChange{Enable: true, Origin: OriginWidgetMenu}); changeError != nil {
		t.Fatalf("unexpected error: %v", changeError)
	}

	if enabled, _ := StartupEnabled(store, exePathWithSpace); !enabled {
		t.Error("expected start with Windows to be on after enabling")
	}
	if store.hasApproval {
		t.Error("expected Thyme's own Task Manager disable to be cleared (author's decision, WE6)")
	}
}

func TestChangeStartup_DisablingDeletesOnlyTheRunValue(t *testing.T) {
	store := &fakeStartupStore{runValue: RunValueFor(exePathWithSpace), hasRunValue: true, approvalBytes: taskManagerDisabled, hasApproval: true}

	if changeError := ChangeStartup(store, exePathWithSpace, StartupChange{Enable: false, Origin: OriginTrayMenu}); changeError != nil {
		t.Fatalf("unexpected error: %v", changeError)
	}

	if store.hasRunValue {
		t.Error("expected the Run value deleted")
	}
	if !store.hasApproval {
		t.Error("expected the approval entry left alone when disabling")
	}
}

func TestChangeStartup_IsIdempotent(t *testing.T) {
	store := &fakeStartupStore{}
	for range 2 {
		if changeError := ChangeStartup(store, exePathWithSpace, StartupChange{Enable: false, Origin: OriginWidgetMenu}); changeError != nil {
			t.Fatalf("disabling twice failed: %v", changeError)
		}
	}
	for range 2 {
		if changeError := ChangeStartup(store, exePathWithSpace, StartupChange{Enable: true, Origin: OriginWidgetMenu}); changeError != nil {
			t.Fatalf("enabling twice failed: %v", changeError)
		}
	}
	if enabled, _ := StartupEnabled(store, exePathWithSpace); !enabled {
		t.Error("expected enabled after enabling twice")
	}
}

// Partial failure: whatever is left behind, the menu reads the registry
// fresh, so it shows the truth rather than what was asked for.
func TestChangeStartup_PartialFailuresAreReportedAndTheReadShowsTheTruth(t *testing.T) {
	testCases := []struct {
		name            string
		store           *fakeStartupStore
		change          StartupChange
		expectedEnabled bool
	}{
		{name: "the Run value cannot be written", store: &fakeStartupStore{failWriteRun: true}, change: StartupChange{Enable: true, Origin: OriginWidgetMenu}, expectedEnabled: false},
		{name: "the Run value is written but the disable cannot be cleared", store: &fakeStartupStore{failDeleteApproval: true, approvalBytes: taskManagerDisabled, hasApproval: true}, change: StartupChange{Enable: true, Origin: OriginWidgetMenu}, expectedEnabled: false},
		{name: "the Run value cannot be deleted", store: &fakeStartupStore{failDeleteRun: true, runValue: RunValueFor(exePathWithSpace), hasRunValue: true}, change: StartupChange{Enable: false, Origin: OriginTrayMenu}, expectedEnabled: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if changeError := ChangeStartup(testCase.store, exePathWithSpace, testCase.change); changeError == nil {
				t.Error("expected the failure to be reported")
			}
			if enabled, _ := StartupEnabled(testCase.store, exePathWithSpace); enabled != testCase.expectedEnabled {
				t.Errorf("expected the fresh read to show %v, got %v", testCase.expectedEnabled, enabled)
			}
		})
	}
}

// Datum 0.14: two surfaces can change this, so the origin is required.
func TestChangeStartup_RefusesAChangeWithoutAnOrigin(t *testing.T) {
	store := &fakeStartupStore{}

	if changeError := ChangeStartup(store, exePathWithSpace, StartupChange{Enable: true}); changeError == nil {
		t.Error("expected a change with no origin to be refused")
	}
	if store.hasRunValue {
		t.Error("expected nothing written for a refused change")
	}
}
