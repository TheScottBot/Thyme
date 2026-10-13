package desktop

import (
	"errors"

	"golang.org/x/sys/windows/registry"

	"github.com/TheScottBot/Thyme/internal/product"
)

// The keys Start with Windows lives in, under HKEY_CURRENT_USER so no
// elevation is needed. StartupApproved is undocumented; what Thyme reads from
// it is recorded in startup.go and the WE6 Plan.
const (
	runKeyPath                = `Software\Microsoft\Windows\CurrentVersion\Run`
	startupApprovedRunKeyPath = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
)

// registryStartupStore is the real StartupStore. The value name is the
// product name (WD2), the same in both keys.
type registryStartupStore struct{}

func openKey(path string, access uint32) (registry.Key, bool, error) {
	key, openError := registry.OpenKey(registry.CURRENT_USER, path, access)
	if errors.Is(openError, registry.ErrNotExist) {
		return 0, false, nil
	}
	if openError != nil {
		return 0, false, openError
	}
	return key, true, nil
}

func (registryStartupStore) ReadRunValue() (string, bool, error) {
	key, exists, openError := openKey(runKeyPath, registry.QUERY_VALUE)
	if openError != nil || !exists {
		return "", false, openError
	}
	defer key.Close()
	value, _, readError := key.GetStringValue(product.Name)
	if errors.Is(readError, registry.ErrNotExist) {
		return "", false, nil
	}
	if readError != nil {
		return "", false, readError
	}
	return value, true, nil
}

func (registryStartupStore) WriteRunValue(value string) error {
	key, _, createError := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if createError != nil {
		return createError
	}
	defer key.Close()
	return key.SetStringValue(product.Name, value)
}

func deleteValueIfPresent(path string) error {
	key, exists, openError := openKey(path, registry.SET_VALUE)
	if openError != nil || !exists {
		return openError
	}
	defer key.Close()
	if deleteError := key.DeleteValue(product.Name); deleteError != nil && !errors.Is(deleteError, registry.ErrNotExist) {
		return deleteError
	}
	return nil
}

func (registryStartupStore) DeleteRunValue() error {
	return deleteValueIfPresent(runKeyPath)
}

func (registryStartupStore) ReadApproval() ([]byte, bool, error) {
	key, exists, openError := openKey(startupApprovedRunKeyPath, registry.QUERY_VALUE)
	if openError != nil || !exists {
		return nil, false, openError
	}
	defer key.Close()
	value, _, readError := key.GetBinaryValue(product.Name)
	if errors.Is(readError, registry.ErrNotExist) {
		return nil, false, nil
	}
	if readError != nil {
		return nil, false, readError
	}
	return value, true, nil
}

func (registryStartupStore) DeleteApproval() error {
	return deleteValueIfPresent(startupApprovedRunKeyPath)
}
