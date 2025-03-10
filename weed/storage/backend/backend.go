package backend

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/chrislusf/seaweedfs/weed/glog"
	"github.com/chrislusf/seaweedfs/weed/pb/master_pb"
	"github.com/chrislusf/seaweedfs/weed/pb/volume_server_pb"
	"github.com/spf13/viper"
)

type BackendStorageFile interface {
	io.ReaderAt
	io.WriterAt
	Truncate(off int64) error
	io.Closer
	GetStat() (datSize int64, modTime time.Time, err error)
	Name() string
}

type BackendStorage interface {
	ToProperties() map[string]string
	NewStorageFile(key string, tierInfo *volume_server_pb.VolumeTierInfo) BackendStorageFile
	CopyFile(f *os.File, fn func(progressed int64, percentage float32) error) (key string, size int64, err error)
}

type StringProperties interface {
	GetString(key string) string
}
type StorageType string
type BackendStorageFactory interface {
	StorageType() StorageType
	BuildStorage(configuration StringProperties, id string) (BackendStorage, error)
}

var (
	BackendStorageFactories = make(map[StorageType]BackendStorageFactory)
	BackendStorages         = make(map[string]BackendStorage)
)

func LoadConfiguration(config *viper.Viper) {

	StorageBackendPrefix := "storage.backend"

	backendSub := config.Sub(StorageBackendPrefix)

	for backendTypeName := range config.GetStringMap(StorageBackendPrefix) {
		backendStorageFactory, found := BackendStorageFactories[StorageType(backendTypeName)]
		if !found {
			glog.Fatalf("backend storage type %s not found", backendTypeName)
		}
		backendTypeSub := backendSub.Sub(backendTypeName)
		for backendStorageId := range backendSub.GetStringMap(backendTypeName) {
			if !backendTypeSub.GetBool(backendStorageId + ".enabled") {
				continue
			}
			backendStorage, buildErr := backendStorageFactory.BuildStorage(backendTypeSub.Sub(backendStorageId), backendStorageId)
			if buildErr != nil {
				glog.Fatalf("fail to create backend storage %s.%s", backendTypeName, backendStorageId)
			}
			BackendStorages[backendTypeName+"."+backendStorageId] = backendStorage
			if backendStorageId == "default" {
				BackendStorages[backendTypeName] = backendStorage
			}
		}
	}

}

func LoadFromPbStorageBackends(storageBackends []*master_pb.StorageBackend) {

	for _, storageBackend := range storageBackends {
		backendStorageFactory, found := BackendStorageFactories[StorageType(storageBackend.Type)]
		if !found {
			glog.Warningf("storage type %s not found", storageBackend.Type)
			continue
		}
		backendStorage, buildErr := backendStorageFactory.BuildStorage(newProperties(storageBackend.Properties), storageBackend.Id)
		if buildErr != nil {
			glog.Fatalf("fail to create backend storage %s.%s", storageBackend.Type, storageBackend.Id)
		}
		BackendStorages[storageBackend.Type+"."+storageBackend.Id] = backendStorage
		if storageBackend.Id == "default" {
			BackendStorages[storageBackend.Type] = backendStorage
		}
	}
}

type Properties struct {
	m map[string]string
}

func newProperties(m map[string]string) *Properties {
	return &Properties{m: m}
}

func (p *Properties) GetString(key string) string {
	if v, found := p.m[key]; found {
		return v
	}
	return ""
}

func ToPbStorageBackends() (backends []*master_pb.StorageBackend) {
	for sName, s := range BackendStorages {
		sType, sId := BackendNameToTypeId(sName)
		if sType == "" {
			continue
		}
		backends = append(backends, &master_pb.StorageBackend{
			Type:       sType,
			Id:         sId,
			Properties: s.ToProperties(),
		})
	}
	return
}

func BackendNameToTypeId(backendName string) (backendType, backendId string) {
	parts := strings.Split(backendName, ".")
	if len(parts) == 1 {
		return backendName, "default"
	}
	if len(parts) != 2 {
		return
	}

	backendType, backendId = parts[0], parts[1]
	return
}
