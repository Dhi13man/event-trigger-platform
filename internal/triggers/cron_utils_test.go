package triggers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateNextFireTime_ValidUTC(t *testing.T) {
	from := time.Date(2025, 1, 2, 3, 4, 0, 0, time.UTC)
	next, err := CalculateNextFireTime("*/5 * * * *", "", from)
	assert.NoError(t, err)
	// Next multiple of 5 minutes: 03:05
	assert.Equal(t, time.Date(2025, 1, 2, 3, 5, 0, 0, time.UTC), next)
}

func TestCalculateNextFireTime_Timezone(t *testing.T) {
	from := time.Date(2025, 1, 2, 7, 0, 0, 0, time.UTC) // 02:00 EST (America/New_York, UTC-5)
	next, err := CalculateNextFireTime("0 3 * * *", "America/New_York", from)
	assert.NoError(t, err)
	// Next 3AM New York should be 08:00 UTC
	assert.Equal(t, time.Date(2025, 1, 2, 8, 0, 0, 0, time.UTC), next)
}

func TestCalculateNextFireTime_InvalidCron(t *testing.T) {
	_, err := CalculateNextFireTime("invalid", "", time.Now())
	assert.Error(t, err)
}

func TestParseCronConfig(t *testing.T) {
	cfg := CronConfig{Cron: "*/10 * * * *", Timezone: "UTC"}
	b, _ := json.Marshal(cfg)
	out, err := ParseCronConfig(b)
	assert.NoError(t, err)
	assert.Equal(t, cfg.Cron, out.Cron)
	assert.Equal(t, cfg.Timezone, out.Timezone)
}

func TestParseCronConfig_MissingCron(t *testing.T) {
	b := []byte(`{"timezone":"UTC"}`)
	_, err := ParseCronConfig(b)
	assert.Error(t, err)
}
