// Copyright (c) 2026 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package apis

import (
	"context"
	"testing"

	"github.com/cloudwego/kitex/client/callopt"
	"github.com/stretchr/testify/assert"

	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/observability/task"
	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/observability/task/taskservice"
)

type fakeTaskClient struct {
	getCalls int
}

func (f *fakeTaskClient) CheckTaskName(ctx context.Context, req *task.CheckTaskNameRequest, callOptions ...callopt.Option) (*task.CheckTaskNameResponse, error) {
	return &task.CheckTaskNameResponse{}, nil
}

func (f *fakeTaskClient) CreateTask(ctx context.Context, req *task.CreateTaskRequest, callOptions ...callopt.Option) (*task.CreateTaskResponse, error) {
	return &task.CreateTaskResponse{}, nil
}

func (f *fakeTaskClient) UpdateTask(ctx context.Context, req *task.UpdateTaskRequest, callOptions ...callopt.Option) (*task.UpdateTaskResponse, error) {
	return &task.UpdateTaskResponse{}, nil
}

func (f *fakeTaskClient) ListTasks(ctx context.Context, req *task.ListTasksRequest, callOptions ...callopt.Option) (*task.ListTasksResponse, error) {
	return &task.ListTasksResponse{}, nil
}

func (f *fakeTaskClient) GetTask(ctx context.Context, req *task.GetTaskRequest, callOptions ...callopt.Option) (*task.GetTaskResponse, error) {
	f.getCalls++
	return &task.GetTaskResponse{}, nil
}

func TestLazyTaskClient_DefersFactoryUntilFirstCall(t *testing.T) {
	factoryCalls := 0
	fake := &fakeTaskClient{}
	factory := func() taskservice.Client {
		factoryCalls++
		return fake
	}

	client := newLazyTaskClient(factory)
	// Factory must not be invoked at construction time.
	assert.Equal(t, 0, factoryCalls)

	_, err := client.GetTask(context.Background(), &task.GetTaskRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 1, factoryCalls)
	assert.Equal(t, 1, fake.getCalls)

	// Subsequent calls reuse the resolved client; factory runs exactly once.
	_, err = client.GetTask(context.Background(), &task.GetTaskRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 1, factoryCalls)
	assert.Equal(t, 2, fake.getCalls)
}

func TestProvideTaskClient_IsLazy(t *testing.T) {
	factoryCalls := 0
	factory := func() taskservice.Client {
		factoryCalls++
		return &fakeTaskClient{}
	}
	_ = provideTaskClient(factory)
	assert.Equal(t, 0, factoryCalls)
}
