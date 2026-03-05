//go:build !cgo

package service

import (
	"fmt"
	"time"
)

type LocalDeckRecommender struct {
}

func NewLocalDeckRecommender(
	masterdataDir string,
	musicmetasData []byte,
	region string,
	algs []string,
	poolSize int,
	timeout time.Duration,
) (*LocalDeckRecommender, error) {
	return nil, fmt.Errorf("LocalDeckRecommender requires CGO_ENABLED=1 but it was disabled during build")
}

func (l *LocalDeckRecommender) Enabled() bool {
	return false
}

func (l *LocalDeckRecommender) Close() {
}

func (l *LocalDeckRecommender) ExpandAlgorithms(option map[string]interface{}) []map[string]interface{} {
	return nil
}

func (l *LocalDeckRecommender) Recommend(req DeckRecommendRequest) (*DeckRecommendResult, error) {
	return nil, fmt.Errorf("CGo is not enabled")
}
