package nsmonitor

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type netNsMonitorClient struct {
	monitor *netNsMonitor
}

func NewClient() networkservice.NetworkServiceClient {
	return &netNsMonitorClient{
		monitor: newMonitor(),
	}
}

func (r *netNsMonitorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	inodeURL, ok := conn.GetMechanism().GetParameters()[common.InodeURL]
	logger := log.FromContext(ctx).WithField("component", "netNsMonitor").WithField(common.InodeURL, inodeURL)
	if ok {
		if result, err := r.monitor.AddNsInode(ctx, inodeURL); err != nil {
			logger.WithField("error", err).Error("unable to monitor")
			return nil, err
		} else {
			logger.Info(result)
		}
	} else {
		logger.Info("inodeURL not found")
	}

	return conn, nil
}

func (r *netNsMonitorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
