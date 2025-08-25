package main

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewECRCollector(t *testing.T) {
	collector := NewECRCollector(nil)

	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	if collector.client != nil {
		t.Error("Expected nil client when passed nil")
	}
}

func TestECRCollectorDescribe(t *testing.T) {
	collector := NewECRCollector(nil)

	ch := make(chan *prometheus.Desc, 10)
	go func() {
		collector.Describe(ch)
		close(ch)
	}()

	var descs []*prometheus.Desc
	for desc := range ch {
		descs = append(descs, desc)
	}

	expectedCount := 9 // Number of metrics we export
	if len(descs) != expectedCount {
		t.Errorf("Expected %d metric descriptions, got %d", expectedCount, len(descs))
	}
}

func TestTimestampLogic(t *testing.T) {
	// Create test images with different timestamps
	now := time.Now()
	older := now.Add(-24 * time.Hour)
	newest := now.Add(-1 * time.Hour)

	// Test data simulating ECR ImageDetail responses
	testImages := []types.ImageDetail{
		{
			ImagePushedAt:        &older,
			LastRecordedPullTime: &older,
		},
		{
			ImagePushedAt:        &now,
			LastRecordedPullTime: &now,
		},
		{
			ImagePushedAt:        &newest,
			LastRecordedPullTime: &newest,
		},
	}

	// Test the timestamp logic (extracted from collectRepositoryMetrics)
	var latestPush, latestPull *time.Time

	for _, image := range testImages {
		// Handle push timestamp - find the latest one
		if image.ImagePushedAt != nil {
			pushTime := *image.ImagePushedAt
			if latestPush == nil || pushTime.After(*latestPush) {
				latestPush = &pushTime
			}
		}

		// Handle pull timestamp - find the latest one
		if image.LastRecordedPullTime != nil {
			pullTime := *image.LastRecordedPullTime
			if latestPull == nil || pullTime.After(*latestPull) {
				latestPull = &pullTime
			}
		}
	}

	// Verify we got the newest timestamps
	if latestPush == nil {
		t.Fatal("Expected to find latest push timestamp")
	}
	if !latestPush.Equal(newest) {
		t.Errorf("Expected latest push to be %v, got %v", newest, *latestPush)
	}

	if latestPull == nil {
		t.Fatal("Expected to find latest pull timestamp")
	}
	if !latestPull.Equal(newest) {
		t.Errorf("Expected latest pull to be %v, got %v", newest, *latestPull)
	}

	t.Logf("✅ Latest push: %v (Unix: %d)", *latestPush, latestPush.Unix())
	t.Logf("✅ Latest pull: %v (Unix: %d)", *latestPull, latestPull.Unix())
}

func TestTimestampLogicWithNilValues(t *testing.T) {
	// Test with nil timestamps
	testImages := []types.ImageDetail{
		{
			ImagePushedAt:        nil,
			LastRecordedPullTime: nil,
		},
	}

	var latestPush, latestPull *time.Time

	for _, image := range testImages {
		if image.ImagePushedAt != nil {
			pushTime := *image.ImagePushedAt
			if latestPush == nil || pushTime.After(*latestPush) {
				latestPush = &pushTime
			}
		}

		if image.LastRecordedPullTime != nil {
			pullTime := *image.LastRecordedPullTime
			if latestPull == nil || pullTime.After(*latestPull) {
				latestPull = &pullTime
			}
		}
	}

	// Should remain nil when no timestamps are present
	if latestPush != nil {
		t.Error("Expected latestPush to remain nil when no timestamps present")
	}
	if latestPull != nil {
		t.Error("Expected latestPull to remain nil when no timestamps present")
	}
}

func TestTimestampLogicWithMixedValues(t *testing.T) {
	now := time.Now()
	older := now.Add(-1 * time.Hour)

	// Test with mixed nil and non-nil timestamps
	testImages := []types.ImageDetail{
		{
			ImagePushedAt:        nil,
			LastRecordedPullTime: &older,
		},
		{
			ImagePushedAt:        &now,
			LastRecordedPullTime: nil,
		},
	}

	var latestPush, latestPull *time.Time

	for _, image := range testImages {
		if image.ImagePushedAt != nil {
			pushTime := *image.ImagePushedAt
			if latestPush == nil || pushTime.After(*latestPush) {
				latestPush = &pushTime
			}
		}

		if image.LastRecordedPullTime != nil {
			pullTime := *image.LastRecordedPullTime
			if latestPull == nil || pullTime.After(*latestPull) {
				latestPull = &pullTime
			}
		}
	}

	// Should find the non-nil values
	if latestPush == nil {
		t.Error("Expected to find push timestamp")
	} else if !latestPush.Equal(now) {
		t.Errorf("Expected push timestamp %v, got %v", now, *latestPush)
	}

	if latestPull == nil {
		t.Error("Expected to find pull timestamp")
	} else if !latestPull.Equal(older) {
		t.Errorf("Expected pull timestamp %v, got %v", older, *latestPull)
	}
}
