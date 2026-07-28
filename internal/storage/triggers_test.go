package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMySQLClient_UpdateTrigger_WhenUpdatesAreInvalid_ThenReturnsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		updates map[string]interface{}
		wantErr string
	}{
		{
			name: "too many fields",
			updates: map[string]interface{}{
				"config":      "{}",
				"name":        "daily report",
				"status":      "active",
				"unsupported": "value",
			},
			wantErr: "too many fields: 4 (maximum 3)",
		},
		{
			name: "unsupported field",
			updates: map[string]interface{}{
				"name = NULL WHERE 1 = 1 --": "value",
			},
			wantErr: `unsupported field "name = NULL WHERE 1 = 1 --"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &MySQLClient{}

			err := client.UpdateTrigger(context.Background(), "trigger-123", tt.updates)

			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestMySQLClient_UpdateTrigger_WhenUpdatesAreEmpty_ThenReturnsNoError(t *testing.T) {
	t.Parallel()
	client := &MySQLClient{}

	err := client.UpdateTrigger(context.Background(), "trigger-123", nil)

	assert.NoError(t, err)
}
