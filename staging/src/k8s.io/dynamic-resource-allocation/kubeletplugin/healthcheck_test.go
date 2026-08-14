/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package kubeletplugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"k8s.io/kubernetes/test/utils/ktesting"
)

func TestHealthCheckServer(t *testing.T) {
	pass := healthCheck{
		name:  "pass",
		check: func(_ context.Context) error { return nil },
	}
	fail := healthCheck{
		name:  "fail",
		check: func(_ context.Context) error { return fmt.Errorf("an error occurred") },
	}

	statusServing := &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}
	statusNotServing := &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
	}

	tests := []struct {
		name             string
		checks           []healthCheck
		service          string
		expectedResponse *grpc_health_v1.HealthCheckResponse
		expectedErr      error
	}{
		{
			name:             "no-checks",
			checks:           nil,
			expectedResponse: statusServing,
		},
		{
			name:             "passing-check",
			checks:           []healthCheck{pass},
			expectedResponse: statusServing,
		},
		{
			name:             "failing-check",
			checks:           []healthCheck{fail},
			expectedResponse: statusNotServing,
		},
		{
			name:             "pass-then-fail",
			checks:           []healthCheck{pass, fail},
			expectedResponse: statusNotServing,
		},
		{
			name:             "fail-then-pass",
			checks:           []healthCheck{fail, pass},
			expectedResponse: statusNotServing,
		},
		{
			name:             "skip-unrequested-services",
			checks:           []healthCheck{pass, fail},
			service:          pass.name,
			expectedResponse: statusServing,
		},
		{
			name: "run-all-checks-sharing-name",
			checks: []healthCheck{
				{name: "svc", check: func(_ context.Context) error { return nil }},
				{name: "svc", check: func(_ context.Context) error { return fmt.Errorf("an error occurred") }},
			},
			service:          "svc",
			expectedResponse: statusNotServing,
		},
		{
			name:        "invalid-requested-service",
			checks:      []healthCheck{pass, fail},
			service:     "not a valid service",
			expectedErr: status.Error(codes.NotFound, `unknown service, valid services are ["" "fail" "pass"]`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			server := healthCheckServer{
				checks: test.checks,
			}
			res, err := server.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: test.service})
			assert.Equal(t, test.expectedResponse, res)
			assert.ErrorIs(t, err, test.expectedErr)
		})
	}
}
