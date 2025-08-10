package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Create a mock handler that embeds the original Handler
type MockHandler struct {
	*Handler
	mock.Mock
}

func (m *MockHandler) validateToken(ctx context.Context, token string) (*Claims, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Claims), args.Error(1)
}

func TestCheckPermissions(t *testing.T) {
	tests := []struct {
		name                   string
		host                   string
		destination            config.Destination
		profiles               map[string]config.Profile
		tenantNames            []string
		mockClaims             *Claims
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
			mockClaims: &Claims{
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
			mockClaims: &Claims{
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
			mockClaims: &Claims{
				Groups: []string{"group2"},
				Email:  "user@example.com",
			},
			expectedError: echo.ErrForbidden,
			expectedEmail: "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("x-id-token", "valid-token")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			c.Set("destination", tt.destination)
			c.Set("tenantNames", tt.tenantNames)

			// Create the real handler
			realHandler := &Handler{
				config: &config.Config{
					Destinations: map[string]config.Destination{
						tt.host: tt.destination,
					},
					Profiles: tt.profiles,
				},
				tokenValidation: true,
			}

			// Create mock handler that embeds the real handler
			mockHandler := &MockHandler{Handler: realHandler}

			// Set up mock expectations
			mockHandler.On("validateToken", mock.Anything, mock.Anything).Return(tt.mockClaims, tt.mockValidateTokenError)

			err := mockHandler.checkPermissions(func(c echo.Context) error {
				return nil
			})(c)

			if tt.expectedError != nil {
				assert.NotNil(t, err)
				assert.IsType(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				mockHandler.AssertExpectations(t)
			}
		})
	}
}
