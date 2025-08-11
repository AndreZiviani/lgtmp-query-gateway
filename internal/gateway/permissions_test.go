package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/oidc/providers/mock"
	oidcTypes "github.com/AndreZiviani/lgtmp-query-gateway/internal/oidc/types"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
)

func TestCheckPermissions(t *testing.T) {
	tests := []struct {
		name                   string
		host                   string
		destination            config.Destination
		profiles               map[string]config.Profile
		tenantNames            []string
		mockClaims             *oidcTypes.Claims
		mockValidateTokenError error
		expectedGroups         []string
		expectedEmail          string
		expectedEnforcedLabels []*labels.Matcher
		expectedError          *echo.HTTPError
	}{
		{
			name: "Valid Request with Allowlist",
			destination: config.Destination{
				Tenants: map[string]config.Tenant{
					"tenant1": {
						Mode:     "allowlist",
						Profiles: []string{"profile1"},
					},
				},
			},
			profiles: map[string]config.Profile{
				"profile1": {
					Groups: []string{"group1"},
				},
			},
			tenantNames: []string{"tenant1"},
			mockClaims: &oidcTypes.Claims{
				Groups: []string{"group1"},
				Email:  "user@example.com",
			},
			expectedGroups: []string{"group1"},
			expectedEmail:  "user@example.com",
		},
		{
			name: "Valid Request with Denylist",
			destination: config.Destination{
				Tenants: map[string]config.Tenant{
					"tenant1": {
						Mode:     "denylist",
						Profiles: []string{"profile1"},
					},
				},
			},
			profiles: map[string]config.Profile{
				"profile1": {
					Groups: []string{"group1"},
				},
			},
			tenantNames: []string{"tenant1"},
			mockClaims: &oidcTypes.Claims{
				Groups: []string{"group2"},
				Email:  "user@example.com",
			},
			expectedGroups: []string{"group2"},
			expectedEmail:  "user@example.com",
		},
		{
			name: "Forbidden Request with Allowlist",
			destination: config.Destination{
				Tenants: map[string]config.Tenant{
					"tenant1": {
						Mode:     "allowlist",
						Profiles: []string{"profile1"},
					},
				},
			},
			profiles: map[string]config.Profile{
				"profile1": {
					Groups: []string{"group1"},
				},
			},
			tenantNames: []string{"tenant1"},
			mockClaims: &oidcTypes.Claims{
				Groups: []string{"group2"},
				Email:  "user@example.com",
			},
			expectedError: echo.ErrForbidden,
			expectedEmail: "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := json.Marshal(tt.mockClaims)
			assert.NoError(t, err)

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("x-id-token", string(token))
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			c.Set("destination", tt.destination)
			c.Set("tenantNames", tt.tenantNames)

			// Create the real handler
			h := &Handler{
				config: &config.Config{
					Destinations: map[string]config.Destination{
						tt.host: tt.destination,
					},
					Profiles: tt.profiles,
				},
				tokenValidation: true,
				provider:        &mock.Provider{},
			}

			err = h.checkPermissions(func(c echo.Context) error {
				return nil
			})(c)

			if tt.expectedError != nil {
				assert.NotNil(t, err)
				assert.IsType(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
