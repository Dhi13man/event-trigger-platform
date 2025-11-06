package triggers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// CronConfig represents the CRON configuration extracted from a trigger.
type CronConfig struct {
	Cron       string                 `json:"cron"`
	Timezone   string                 `json:"timezone,omitempty"`
	Endpoint   string                 `json:"endpoint"`
	HTTPMethod string                 `json:"http_method"`
	Headers    map[string]string      `json:"headers,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
}

// CalculateNextFireTime calculates the next fire time for a CRON expression.
// This function is shared between TriggerService (for trigger creation)
// and Scheduler (for calculating next occurrence after firing).
//
// Parameters:
//   - cronExpr: CRON expression (e.g., "0 9 * * *" for daily at 9am)
//   - timezone: Timezone name (e.g., "America/New_York"), empty string defaults to UTC
//   - from: Calculate next fire time from this timestamp
//
// Returns:
//   - Next fire time in UTC
//   - Error if CRON expression is invalid or timezone is invalid
func CalculateNextFireTime(cronExpr string, timezone string, from time.Time) (time.Time, error) {
	// Resolve timezone
	loc, err := resolveTimezone(timezone)
	if err != nil {
		return time.Time{}, err
	}

	// Parse CRON expression
	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Calculate next run time in the specified timezone, then convert to UTC
	nextRun := schedule.Next(from.In(loc)).UTC()
	return nextRun, nil
}

// ParseCronConfig extracts CRON configuration from a trigger's config JSON.
// This is used by the scheduler to get CRON expression and timezone for calculating next fire time.
func ParseCronConfig(configJSON json.RawMessage) (*CronConfig, error) {
	var config CronConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to parse cron config: %w", err)
	}

	if config.Cron == "" {
		return nil, fmt.Errorf("cron expression is required")
	}

	return &config, nil
}

// resolveTimezone resolves a timezone string to a time.Location.
// Empty string defaults to UTC.
func resolveTimezone(tz string) (*time.Location, error) {
	if tz == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", tz, err)
	}
	return loc, nil
}
