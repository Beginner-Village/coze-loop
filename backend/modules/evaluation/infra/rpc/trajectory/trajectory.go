// Copyright (c) 2025 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package trajectory

import (
	"context"
	"sync"

	"github.com/bytedance/gg/gcond"
	"github.com/bytedance/gg/gptr"

	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/observability/observabilitytraceservice"
	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/observability/trace"
	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/trajectory"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/component/rpc"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/entity"
	gslice "github.com/coze-dev/coze-loop/backend/pkg/lang/slices"
	"github.com/coze-dev/coze-loop/backend/pkg/logs"
)

func NewAdapter(tracer func() observabilitytraceservice.Client) rpc.ITrajectoryAdapter {
	return &adapterImpl{
		tracerFactory: tracer,
	}
}

type adapterImpl struct {
	tracerFactory func() observabilitytraceservice.Client

	tracer observabilitytraceservice.Client
	mu     sync.Mutex
}

func (a *adapterImpl) GetTracer() observabilitytraceservice.Client {
	if a.tracer != nil {
		return a.tracer
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tracer == nil {
		a.tracer = a.tracerFactory()
	}
	return a.tracer
}

func (a *adapterImpl) ListTrajectory(ctx context.Context, spaceID int64, traceIDs []string, startTimeMS *int64) ([]*entity.Trajectory, error) {
	const PlatformType = "default"
	req := &trace.ListTrajectoryRequest{
		PlatformType: PlatformType,
		WorkspaceID:  spaceID,
		TraceIds:     traceIDs,
		StartTime:    startTimeMS,
	}
	resp, err := a.GetTracer().ListTrajectory(ctx, req)
	if err != nil {
		return nil, err
	}

	// 去日志:resp.Trajectories 含链路对话/会话原文,仅打印请求标识与返回数量,不打印明文
	logs.CtxInfo(ctx, "ListTrajectory success, space_id: %v, trace_ids: %v, trajectory_count: %v",
		spaceID, traceIDs, len(resp.GetTrajectories()))

	return gslice.Transform(resp.GetTrajectories(), func(e *trajectory.Trajectory, _ int) *entity.Trajectory {
		return gcond.IfLazyR(e == nil, nil, func() *entity.Trajectory { return gptr.Of(entity.Trajectory(*e)) })
	}), nil
}
