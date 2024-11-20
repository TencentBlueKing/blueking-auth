package handler

import (
	"testing"

	"bkauth/pkg/api/common"

	"github.com/stretchr/testify/assert"
)

func TestCreateAppSerializer_Validate(t *testing.T) {
	tests := []struct {
		name       string
		serializer createAppSerializer
		wantErr    bool
	}{
		{
			name: "invalid tenant_id",
			serializer: createAppSerializer{
				TenantID: "1invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid app_code",
			serializer: createAppSerializer{
				TenantID: "validtenant",
				AppCodeSerializer: common.AppCodeSerializer{
					AppCode: "invalid app code",
				},
			},
			wantErr: true,
		},
		{
			name: "all valid",
			serializer: createAppSerializer{
				TenantID: "validtenant",
				AppCodeSerializer: common.AppCodeSerializer{
					AppCode: "validappcode",
				},
			},
			wantErr: false,
		},
		{
			name: "tenant_id is *",
			serializer: createAppSerializer{
				TenantID: "*",
				AppCodeSerializer: common.AppCodeSerializer{
					AppCode: "validappcode",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.serializer.validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
