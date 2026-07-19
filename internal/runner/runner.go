package runner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"manga-drama-studio/internal/domain"
	"manga-drama-studio/internal/store"
	"manga-drama-studio/internal/workflow"
)

type Runner struct {
	store    store.Repository
	executor Executor
	broker   *Broker
	mu       sync.Mutex
	cancels  map[string]context.CancelFunc
}

func New(data store.Repository, executor Executor, broker *Broker) *Runner {
	return &Runner{store: data, executor: executor, broker: broker, cancels: map[string]context.CancelFunc{}}
}

func (r *Runner) Start(run domain.Run) error {
	if err := r.store.SaveRun(context.Background(), run); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.cancels[run.ID] = cancel
	r.mu.Unlock()
	go r.execute(ctx, run.ID)
	return nil
}

func (r *Runner) Cancel(runID string) error {
	if _, err := r.store.Run(context.Background(), runID); err != nil {
		return err
	}
	r.mu.Lock()
	cancel := r.cancels[runID]
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (r *Runner) Resume(runID string) error {
	run, err := r.store.Run(context.Background(), runID)
	if err != nil {
		return err
	}
	if run.Status != domain.RunFailed && run.Status != domain.RunCancelled && run.Status != domain.RunInterrupted {
		return errors.New("仅失败、取消或中断的任务可以恢复")
	}
	run.Status = domain.RunQueued
	run.Message = "任务已恢复并重新入队"
	run.FinishedAt = nil
	run.InterruptedAt = nil
	return r.Start(run)
}

func (r *Runner) execute(ctx context.Context, runID string) {
	defer func() { r.mu.Lock(); delete(r.cancels, runID); r.mu.Unlock() }()
	run, err := r.store.Run(ctx, runID)
	if err != nil {
		return
	}
	order, err := workflow.TopologicalOrder(run.Workflow)
	if err != nil {
		r.finish(runID, domain.RunFailed, err.Error())
		return
	}
	now := time.Now().UTC()
	r.update(runID, "run", "", domain.RunRunning, 0, "工作流开始执行", func(run *domain.Run) { run.StartedAt = &now })
	outputs := map[string]any{}
	for index, node := range order {
		if ctx.Err() != nil {
			r.finish(runID, domain.RunCancelled, "用户已取消")
			return
		}
		started := time.Now().UTC()
		r.update(runID, "node", node.ID, domain.RunRunning, 0, "节点开始", func(run *domain.Run) {
			nodeRun := run.Nodes[node.ID]
			nodeRun.Status = domain.RunRunning
			nodeRun.StartedAt = &started
			nodeRun.Message = "节点开始"
			run.Nodes[node.ID] = nodeRun
		})
		executionContext := context.WithValue(ctx, executionInfoKey{}, executionInfo{RunID: run.ID, ProjectID: run.ProjectID, NodeID: node.ID})
		nodeOutputs, execErr := r.executor.Execute(executionContext, node, outputs, func(progress int, message string) {
			overall := (index*100 + progress) / len(order)
			r.update(runID, "node", node.ID, domain.RunRunning, overall, message, func(run *domain.Run) {
				nodeRun := run.Nodes[node.ID]
				nodeRun.Progress = progress
				nodeRun.Message = message
				run.Nodes[node.ID] = nodeRun
			})
		})
		if execErr != nil {
			if errors.Is(execErr, context.Canceled) {
				r.finish(runID, domain.RunCancelled, "用户已取消")
				return
			}
			r.update(runID, "node", node.ID, domain.RunFailed, run.Progress, execErr.Error(), func(run *domain.Run) {
				finished := time.Now().UTC()
				nodeRun := run.Nodes[node.ID]
				nodeRun.Status = domain.RunFailed
				nodeRun.Message = execErr.Error()
				nodeRun.FinishedAt = &finished
				run.Nodes[node.ID] = nodeRun
			})
			r.finish(runID, domain.RunFailed, fmt.Sprintf("节点 %s 执行失败", node.Name))
			return
		}
		for key, value := range nodeOutputs {
			outputs[node.ID+"."+key] = value
		}
		finished := time.Now().UTC()
		r.update(runID, "node", node.ID, domain.RunSucceeded, ((index+1)*100)/len(order), "节点完成", func(run *domain.Run) {
			nodeRun := run.Nodes[node.ID]
			nodeRun.Status = domain.RunSucceeded
			nodeRun.Progress = 100
			nodeRun.Message = "节点完成"
			nodeRun.Outputs = nodeOutputs
			nodeRun.FinishedAt = &finished
			run.Nodes[node.ID] = nodeRun
		})
	}
	r.finish(runID, domain.RunSucceeded, "漫剧工作流生成完成")
}

func (r *Runner) finish(runID string, status domain.RunStatus, message string) {
	finished := time.Now().UTC()
	progress := 100
	if status != domain.RunSucceeded {
		current, _ := r.store.Run(context.Background(), runID)
		progress = current.Progress
	}
	r.update(runID, "run", "", status, progress, message, func(run *domain.Run) { run.FinishedAt = &finished })
}

func (r *Runner) update(runID, eventType, nodeID string, status domain.RunStatus, progress int, message string, mutate func(*domain.Run)) {
	_, err := r.store.UpdateRun(context.Background(), runID, func(run *domain.Run) {
		if eventType == "run" {
			run.Status = status
		}
		run.Progress = progress
		run.Message = message
		mutate(run)
	})
	if err != nil {
		return
	}
	r.broker.Publish(domain.RunEvent{Type: eventType, RunID: runID, NodeID: nodeID, Status: status, Progress: progress, Message: message, Timestamp: time.Now().UTC()})
}
