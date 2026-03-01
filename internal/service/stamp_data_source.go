package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type StampRecord struct {
	ID              int    `json:"id"`
	AssetbundleName string `json:"assetbundleName"`
}

type StampDataSource interface {
	DefaultRegion() string
	GetStamps() ([]StampRecord, error)
}

type MasterDataStampSource struct {
	region  string
	dataDir string

	mu     sync.RWMutex
	loaded bool
	stamps []StampRecord
}

func NewMasterDataStampSource(svc *MasterDataService) *MasterDataStampSource {
	if svc == nil {
		return nil
	}
	return &MasterDataStampSource{
		region:  svc.GetRegion(),
		dataDir: svc.GetDataDir(),
	}
}

func (m *MasterDataStampSource) DefaultRegion() string {
	return m.region
}

func (m *MasterDataStampSource) GetStamps() ([]StampRecord, error) {
	m.mu.RLock()
	if m.loaded {
		out := append([]StampRecord(nil), m.stamps...)
		m.mu.RUnlock()
		return out, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.loaded {
		path := filepath.Join(m.dataDir, "stamps.json")
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read stamps.json failed: %w", err)
		}
		var stamps []StampRecord
		if err := json.Unmarshal(data, &stamps); err != nil {
			return nil, fmt.Errorf("parse stamps.json failed: %w", err)
		}
		m.stamps = stamps
		m.loaded = true
	}
	return append([]StampRecord(nil), m.stamps...), nil
}
