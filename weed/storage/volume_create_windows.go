// +build windows

package storage

import (
	"github.com/chrislusf/seaweedfs/weed/storage/backend/memory_map"
	"golang.org/x/sys/windows"

	"github.com/chrislusf/seaweedfs/weed/glog"
	"github.com/chrislusf/seaweedfs/weed/storage/backend"
	"github.com/chrislusf/seaweedfs/weed/storage/backend/memory_map/os_overloads"
)

func createVolumeFile(fileName string, preallocate int64, memoryMapSizeMB uint32) (backend.BackendStorageFile, error) {

	if preallocate > 0 {
		glog.V(0).Infof("Preallocated disk space for %s is not supported", fileName)
	}

	if memoryMapSizeMB > 0 {
		file, e := os_overloads.OpenFile(fileName, windows.O_RDWR|windows.O_CREAT, 0644, true)
		return memory_map.NewMemoryMappedFile(file, memoryMapSizeMB), e
	} else {
		file, e := os_overloads.OpenFile(fileName, windows.O_RDWR|windows.O_CREAT|windows.O_TRUNC, 0644, false)
		return backend.NewDiskFile(file), e
	}

}
