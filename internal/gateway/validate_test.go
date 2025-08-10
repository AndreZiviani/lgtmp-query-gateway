package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name                string
		host                string
		tenantID            string
		expectedTenantNames []string
		expectedDestion     *config.Destination
		expectedError       *echo.HTTPError
	}{
		{
			name:                "Valid Request",
			host:                "example.com",
			tenantID:            "tenant1",
			expectedTenantNames: []string{"tenant1"},
			expectedDestion: &config.Destination{
				Upstream: "http://upstream.example.com",
			},
			expectedError: nil,
		},
		{
			name:                "Missing Tenant ID",
			host:                "example.com",
			tenantID:            "",
			expectedTenantNames: nil,
			expectedDestion: &config.Destination{
				Upstream: "http://upstream.example.com",
			},
			expectedError: echo.ErrBadRequest,
		},
		{
			name:                "Destination Not Found",
			host:                "unknown.com",
			tenantID:            "tenant1",
			expectedTenantNames: nil,
			expectedDestion:     nil,
			expectedError:       echo.ErrNotFound,
		},
		{
			name:                "Multi-Tenant Request",
			host:                "example.com",
			tenantID:            "tenant1|tenant2",
			expectedTenantNames: nil,
			expectedDestion: &config.Destination{
				Upstream: "http://upstream.example.com",
			},
			expectedError: echo.ErrNotImplemented,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(TenantIDHeader, tt.tenantID)
			req.Host = tt.host
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &Handler{
				config: &config.Config{
					Destinations: map[string]config.Destination{},
				},
			}

			if tt.expectedDestion != nil {
				h.config.Destinations[tt.host] = *tt.expectedDestion
			}

			err := h.validateRequest(func(c echo.Context) error {
				return nil
			})(c)

			if tt.expectedError != nil {
				assert.NotNil(t, err)
				assert.IsType(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedTenantNames != nil {
				assert.NotNil(t, c.Get("tenantNames"))
				assert.Equal(t, tt.expectedTenantNames, c.Get("tenantNames").([]string))
			} else {
				assert.Nil(t, c.Get("tenantNames"))
			}
		})
	}
}
