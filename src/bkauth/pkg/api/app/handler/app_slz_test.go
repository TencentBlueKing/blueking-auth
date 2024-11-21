package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"bkauth/pkg/api/common"
	"bkauth/pkg/util"
)

func TestCreateAppSerializer_Validate(t *testing.T) {
	tests := []struct {
		name       string
		serializer createAppSerializer
		wantErr    bool
		errMsg     string
	}{
		{
			name: "tenant_type=global and tenant_id not empty",
			serializer: createAppSerializer{
				Tenant: tenantSerializer{
					Type: util.TenantTypeGlobal,
					ID:   "some_id",
				},
			},
			wantErr: true,
			errMsg:  "tenant_id should be empty when tenant_type is global",
		},
		{
			name: "tenant_type=single and tenant_id not valid",
			serializer: createAppSerializer{
				Tenant: tenantSerializer{
					Type: util.TenantTypeSingle,
					ID:   "123",
				},
			},
			wantErr: true,
			errMsg:  common.ErrInvalidTenantID.Error(),
		},
		{
			name: "tenant_id tenant_type valid, but app_code not valid",
			serializer: createAppSerializer{
				Tenant: tenantSerializer{
					Type: util.TenantTypeSingle,
					ID:   "valid-id",
				},
				AppCodeSerializer: common.AppCodeSerializer{
					AppCode: "==1",
				},
			},
			wantErr: true,
			errMsg:  common.ErrInvalidAppCode.Error(),
		},
		{
			name: "all valid",
			serializer: createAppSerializer{
				Tenant: tenantSerializer{
					Type: util.TenantTypeSingle,
					ID:   "valid-id",
				},
				AppCodeSerializer: common.AppCodeSerializer{
					AppCode: "valid_app_code",
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
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}