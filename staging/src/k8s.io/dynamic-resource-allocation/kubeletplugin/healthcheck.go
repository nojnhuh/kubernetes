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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
)

type healthCheckServer struct {
	grpc_health_v1.UnimplementedHealthServer

	checks []healthCheck
}

type healthCheck struct {
	name  string
	check func(ctx context.Context) error
}

// Check implements [grpc_health_v1.HealthServer.Check].
func (h *healthCheckServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	logger := klog.FromContext(ctx)

	// The empty string requests the overall health. Here, that means the
	// aggregate health of all the registered checks.
	// https://github.com/grpc/grpc/blob/838ece08491e32cdbbd83360e0637f2543eede99/doc/health-checking.md
	const allServices = ""

	validServices := make(sets.Set[string], len(h.checks)+1)
	validServices.Insert(allServices)
	for _, check := range h.checks {
		validServices.Insert(check.name)
	}
	if !validServices.Has(req.GetService()) {
		return nil, status.Errorf(codes.NotFound, "unknown service, valid services are %q", sets.List(validServices))
	}

	status := &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
	}

	for _, check := range h.checks {
		if s := req.GetService(); s != "" && s != check.name {
			continue
		}
		if err := check.check(ctx); err != nil {
			logger.Error(err, "check failed", "check", check.name)
			return status, nil
		} else {
			logger.V(6).Info("check passed", "check", check.name)
		}
	}

	status.Status = grpc_health_v1.HealthCheckResponse_SERVING
	return status, nil
}
