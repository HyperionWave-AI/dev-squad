package agents

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgentRole_Validate(t *testing.T) {
	tests := []struct {
		name    string
		role    AgentRole
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid agent role",
			role: AgentRole{
				ID:          "test-agent-1",
				Name:        "Test Agent",
				Description: "A test agent",
				Category:    "test",
				Tools:       []string{"tool1", "tool2"},
				Capabilities: []string{"cap1", "cap2"},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			role: AgentRole{
				Name:        "Test Agent",
				Description: "A test agent",
				Category:    "test",
			},
			wantErr: true,
			errMsg:  "ID is required",
		},
		{
			name: "missing name",
			role: AgentRole{
				ID:          "test-agent-1",
				Description: "A test agent",
				Category:    "test",
			},
			wantErr: true,
			errMsg:  "Name is required",
		},
		{
			name: "missing category",
			role: AgentRole{
				ID:          "test-agent-1",
				Name:        "Test Agent",
				Description: "A test agent",
			},
			wantErr: true,
			errMsg:  "Category is required",
		},
		{
			name: "empty tools is valid",
			role: AgentRole{
				ID:          "test-agent-1",
				Name:        "Test Agent",
				Description: "A test agent",
				Category:    "test",
				Tools:       []string{},
			},
			wantErr: false,
		},
		{
			name: "nil tools is valid",
			role: AgentRole{
				ID:          "test-agent-1",
				Name:        "Test Agent",
				Description: "A test agent",
				Category:    "test",
				Tools:       nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.role.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAgentRole_IsActive(t *testing.T) {
	role := AgentRole{
		ID:       "test-agent-1",
		Name:     "Test Agent",
		Category: "test",
		Status:   StatusActive,
	}

	assert.True(t, role.IsActive())

	role.Status = StatusInactive
	assert.False(t, role.IsActive())

	role.Status = ""
	assert.False(t, role.IsActive())
}

func TestAgentRole_HasTool(t *testing.T) {
	role := AgentRole{
		ID:       "test-agent-1",
		Name:     "Test Agent",
		Category: "test",
		Tools:    []string{"tool1", "tool2", "tool3"},
	}

	assert.True(t, role.HasTool("tool1"))
	assert.True(t, role.HasTool("tool2"))
	assert.False(t, role.HasTool("tool4"))
	assert.False(t, role.HasTool(""))

	role.Tools = nil
	assert.False(t, role.HasTool("tool1"))

	role.Tools = []string{}
	assert.False(t, role.HasTool("tool1"))
}

func TestAgentRole_HasCapability(t *testing.T) {
	role := AgentRole{
		ID:            "test-agent-1",
		Name:          "Test Agent",
		Category:      "test",
		Capabilities: []string{"cap1", "cap2", "cap3"},
	}

	assert.True(t, role.HasCapability("cap1"))
	assert.True(t, role.HasCapability("cap2"))
	assert.False(t, role.HasCapability("cap4"))
	assert.False(t, role.HasCapability(""))

	role.Capabilities = nil
	assert.False(t, role.HasCapability("cap1"))

	role.Capabilities = []string{}
	assert.False(t, role.HasCapability("cap1"))
}

func TestAgentRole_SetDefaults(t *testing.T) {
	role := AgentRole{
		ID:   "test-agent-1",
		Name: "Test Agent",
	}

	role.SetDefaults()

	assert.Equal(t, StatusActive, role.Status)
	assert.NotNil(t, role.Tools)
	assert.NotNil(t, role.Capabilities)
	assert.NotNil(t, role.CreatedAt)
	assert.NotNil(t, role.UpdatedAt)
}

func TestAgentRole_UpdateTimestamp(t *testing.T) {
	role := AgentRole{
		ID:        "test-agent-1",
		Name:      "Test Agent",
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	oldTime := role.UpdatedAt
	role.UpdateTimestamp()

	assert.True(t, role.UpdatedAt.After(oldTime))
}

func TestAgentInstance_Validate(t *testing.T) {
	tests := []struct {
		name    string
		instance AgentInstance
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid instance",
			instance: AgentInstance{
				ID:     "instance-1",
				RoleID: "role-1",
				Status: InstanceStatusActive,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			instance: AgentInstance{
				RoleID: "role-1",
				Status: InstanceStatusActive,
			},
			wantErr: true,
			errMsg:  "ID is required",
		},
		{
			name: "missing RoleID",
			instance: AgentInstance{
				ID:     "instance-1",
				Status: InstanceStatusActive,
			},
			wantErr: true,
			errMsg:  "RoleID is required",
		},
		{
			name: "empty status",
			instance: AgentInstance{
				ID:     "instance-1",
				RoleID: "role-1",
			},
			wantErr: true,
			errMsg:  "Status is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.instance.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAgentInstance_IsActive(t *testing.T) {
	instance := AgentInstance{
		ID:     "instance-1",
		RoleID: "role-1",
		Status: InstanceStatusActive,
	}

	assert.True(t, instance.IsActive())

	instance.Status = InstanceStatusInactive
	assert.False(t, instance.IsActive())

	instance.Status = InstanceStatusError
	assert.False(t, instance.IsActive())
}

func TestAgentInstance_SetDefaults(t *testing.T) {
	instance := AgentInstance{
		ID:     "instance-1",
		RoleID: "role-1",
	}

	instance.SetDefaults()

	assert.Equal(t, InstanceStatusActive, instance.Status)
	assert.NotNil(t, instance.CreatedAt)
	assert.NotNil(t, instance.UpdatedAt)
	assert.NotNil(t, instance.LastSeenAt)
}

func TestAgentInstance_UpdateTimestamp(t *testing.T) {
	instance := AgentInstance{
		ID:        "instance-1",
		RoleID:    "role-1",
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	oldTime := instance.UpdatedAt
	instance.UpdateTimestamp()

	assert.True(t, instance.UpdatedAt.After(oldTime))
}

func TestAgentInstance_UpdateLastSeen(t *testing.T) {
	instance := AgentInstance{
		ID:         "instance-1",
		RoleID:     "role-1",
		LastSeenAt: time.Now().Add(-time.Hour),
	}

	oldTime := instance.LastSeenAt
	instance.UpdateLastSeen()

	assert.True(t, instance.LastSeenAt.After(oldTime))
}

func TestStatusConfigurations(t *testing.T) {
	// Test all status constants
	assert.Equal(t, "active", StatusActive)
	assert.Equal(t, "inactive", StatusInactive)
	assert.Equal(t, "error", StatusError)
	assert.Equal(t, "maintenance", StatusMaintenance)

	// Test instance status constants
	assert.Equal(t, "active", InstanceStatusActive)
	assert.Equal(t, "inactive", InstanceStatusInactive)
	assert.Equal(t, "error", InstanceStatusError)
}