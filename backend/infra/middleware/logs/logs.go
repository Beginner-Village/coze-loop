// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package logs

import (
	"context"
	"errors"
	"slices"

	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/pkg/kerrors"
	"github.com/cloudwego/kitex/pkg/rpcinfo"

	"github.com/coze-dev/coze-loop/backend/pkg/json"
	"github.com/coze-dev/coze-loop/backend/pkg/logs"
)

const (
	extraKeyAffectStability = "biz_err_affect_stability"
)

// simplifyMethods 列出会携带对话/评测原文(span input/output 等)的高敏感方法,
// 对这些方法的流量日志一律不打印完整 req/resp 明文,避免敏感内容落盘。
var simplifyMethods = []string{
	// observability: trace 上报与读取, span input/output 即对话/模型调用原文
	"IngestTraces", "OtelIngestTraces", "IngestTracesInner",
	"ListSpans", "ListPreSpan", "GetTrace", "SearchTraceTree",
	"SearchTraceOApi", "SearchTraceTreeOApi", "ListSpansOApi", "ListPreSpanOApi", "ListTracesOApi",
	"ListTraceChat", "ListThreadChat", "ListTrajectory", "ExtractSpanInfo",
	"PreviewExportTracesToDataset",
	// observability: annotation 携带人工标注/修正文本(可能含原文片段)
	"CreateManualAnnotation", "UpdateManualAnnotation", "ListAnnotations", "ListWorkspaceAnnotations",
	"CreateAnnotation",
	// llm runtime: 完整对话消息
	"Chat", "ChatStream",
	// prompt 执行/调试: 变量与消息原文, resp 含模型输出; mget 返回 prompt 模板正文(SDK 热路径)
	"Execute", "ExecuteStreaming", "ExecuteInternal", "DebugStreaming",
	"SaveDebugContext", "GetDebugContext", "ListDebugHistory",
	"BatchGetPromptByPromptKey",
	// evaluation: 评测目标执行/调试与记录读取, 含被测对象输入输出原文
	"ExecuteEvalTarget", "AsyncExecuteEvalTarget", "DebugEvalTarget", "AsyncDebugEvalTarget",
	"MockEvalTargetOutput", "GetEvalTargetRecord", "BatchGetEvalTargetRecords",
	"GetEvalTargetOutputFieldContent", "GetEvalTargetOutputFieldContentOApi",
	// evaluation SPI/回报: 调用与结果上报携带评测输入输出原文
	"InvokeEvalTarget", "AsyncInvokeEvalTarget", "InvokeEvaluator",
	"ReportEvalTargetInvokeResult", "ReportEvaluatorInvokeResult",
	// evaluation: 评估器运行/调试与记录, 含评测输入与评估输出原文
	"RunEvaluator", "DebugEvaluator", "BatchDebugEvaluator", "AsyncRunEvaluator", "AsyncDebugEvaluator",
	"DebugBuiltinEvaluator", "RunEvaluatorOApi", "RunBuiltinEvaluatorOApi",
	"UpdateEvaluatorRecord", "CorrectEvaluatorRecordOApi",
	"GetEvaluatorRecord", "BatchGetEvaluatorRecords", "BatchGetEvaluatorRecordsOApi",
	// evaluation: 评测集条目读写, item 字段即评测数据原文
	"BatchCreateEvaluationSetItems", "UpdateEvaluationSetItem", "BatchGetEvaluationSetItems",
	"ListEvaluationSetItems", "GetEvaluationSetItemField",
	"BatchCreateEvaluationSetItemsOApi", "BatchUpdateEvaluationSetItemsOApi",
	"ListEvaluationSetVersionItemsOApi", "GetEvaluationItemFieldOApi",
	// data: 数据集条目读写, 同为数据原文
	"BatchCreateDatasetItems", "UpdateDatasetItem", "GetDatasetItem", "BatchGetDatasetItems",
	"BatchGetDatasetItemsByVersion", "ListDatasetItems", "ListDatasetItemsByVersion",
	// evaluation: 实验结果读取, 含 target/evaluator 输出原文
	"BatchGetExperimentResult", "ListExperimentResultOApi",
}

// dump 在非敏感方法上返回 req/resp 的 JSON,在敏感方法(simplifyMethods)上返回掩码占位符,
// 防止把对话/评测原文打到日志。
func dump(to string, v any) string {
	if slices.Contains(simplifyMethods, to) {
		return "-"
	}
	return json.Jsonify(v)
}

func LogTrafficMW(next endpoint.Endpoint) endpoint.Endpoint {
	disabled := func() bool {
		return logs.DefaultLogger().GetLevel() > logs.InfoLevel
	}

	return func(ctx context.Context, req, resp any) (err error) {
		var (
			info   = rpcinfo.GetRPCInfo(ctx)
			bizErr kerrors.BizStatusErrorIface
			to     = "unknown"
		)
		if info != nil && info.To() != nil {
			to = info.To().Method()
		}

		logs.CtxInfo(ctx, "[%s] start", to)
		defer func() {
			logs.CtxInfo(ctx, "[%s] end", to)
		}()
		err = next(ctx, req, resp)
		if err == nil && disabled() {
			return err
		}

		if info != nil && info.Invocation() != nil && info.Invocation().BizStatusErr() != nil {
			bizErr = info.Invocation().BizStatusErr()
		}
		if bizErr == nil && err != nil {
			errors.As(err, &bizErr)
		}

		switch {
		case err != nil && bizErr == nil:
			logs.CtxError(ctx, "RPC %s failed, req=%s, err=%v", to, dump(to, req), err)

		case bizErr != nil:
			reqStr, respStr := dump(to, req), dump(to, resp)
			if v := bizErr.BizExtra()[extraKeyAffectStability]; v == "1" {
				logs.CtxError(ctx, "RPC %s failed, req=%s, biz_err=%+v, resp=%s", to, reqStr, bizErr, respStr)
			} else {
				logs.CtxWarn(ctx, "RPC %s failed, req=%s, biz_err=%+v, resp=%s", to, reqStr, bizErr, respStr)
			}

		default:
			if logs.DefaultLogger().GetLevel() <= logs.DebugLevel {
				logs.CtxDebug(ctx, "RPC %s succeeded, req=%s, resp=%s", to, dump(to, req), dump(to, resp))
			}
		}

		return err
	}
}
