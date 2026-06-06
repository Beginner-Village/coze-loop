// Copyright (c) 2026 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package apis

import (
	"context"
	"sync"

	"github.com/cloudwego/kitex/client/callopt"

	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/observability/task"
	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/observability/task/taskservice"
)

// lazyTaskClient defers resolving the underlying taskservice.Client until the
// first method call.
//
// Why this exists: provideTaskClient is invoked eagerly during
// InitEvaluationHandler, but at that point the observability handler (which
// owns the real ITaskApplication backing the task client factory) has not been
// constructed yet — InitObservabilityHandler runs afterwards. Calling the
// factory eagerly therefore captures a task client wrapping a nil
// ITaskApplication, so every task call made through the evaluation handler
// would no-op against a dead impl instead of reaching the live application.
//
// By deferring factory() to first use (request time), the factory resolves the
// fully-initialised observability handler. sync.Once guarantees the factory is
// invoked exactly once, so we don't re-create / re-consume the underlying
// client on every call.
type lazyTaskClient struct {
	factory func() taskservice.Client
	once    sync.Once
	client  taskservice.Client
}

func newLazyTaskClient(factory func() taskservice.Client) taskservice.Client {
	return &lazyTaskClient{factory: factory}
}

func (l *lazyTaskClient) resolve() taskservice.Client {
	l.once.Do(func() {
		l.client = l.factory()
	})
	return l.client
}

func (l *lazyTaskClient) CheckTaskName(ctx context.Context, req *task.CheckTaskNameRequest, callOptions ...callopt.Option) (*task.CheckTaskNameResponse, error) {
	return l.resolve().CheckTaskName(ctx, req, callOptions...)
}

func (l *lazyTaskClient) CreateTask(ctx context.Context, req *task.CreateTaskRequest, callOptions ...callopt.Option) (*task.CreateTaskResponse, error) {
	return l.resolve().CreateTask(ctx, req, callOptions...)
}

func (l *lazyTaskClient) UpdateTask(ctx context.Context, req *task.UpdateTaskRequest, callOptions ...callopt.Option) (*task.UpdateTaskResponse, error) {
	return l.resolve().UpdateTask(ctx, req, callOptions...)
}

func (l *lazyTaskClient) ListTasks(ctx context.Context, req *task.ListTasksRequest, callOptions ...callopt.Option) (*task.ListTasksResponse, error) {
	return l.resolve().ListTasks(ctx, req, callOptions...)
}

func (l *lazyTaskClient) GetTask(ctx context.Context, req *task.GetTaskRequest, callOptions ...callopt.Option) (*task.GetTaskResponse, error) {
	return l.resolve().GetTask(ctx, req, callOptions...)
}
