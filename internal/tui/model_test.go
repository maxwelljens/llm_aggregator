package tui

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/runtime"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_ExecuteResult(t *testing.T) {
	tests := []struct {
		name        string
		result      runtime.Result
		err         error
		wantSummary string
	}{
		{
			name:        "success captures summary",
			result:      runtime.Result{Summary: "the summary"},
			wantSummary: "the summary",
		},
		{
			name:   "error is captured",
			result: runtime.Result{},
			err:    errors.New("boom"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := func(ctx context.Context) (runtime.Result, error) {
				return tt.result, tt.err
			}
			m := New(exec, context.Background())

			msg := m.startProcessing()
			model, cmd := m.Update(msg)

			gotResult, gotErr, finished := model.(*Model).FinalResult()
			if !finished {
				t.Fatal("processing did not finish")
			}
			if gotErr != tt.err {
				t.Errorf("error = %v, want %v", gotErr, tt.err)
			}
			if gotResult.Summary != tt.wantSummary {
				t.Errorf("summary = %q, want %q", gotResult.Summary, tt.wantSummary)
			}
			if cmd != nil {
				t.Errorf("command = %v, want nil (program stays alive after completion)", cmd)
			}
		})
	}
}

func TestModel_NotFinishedBeforeExecution(t *testing.T) {
	m := New(func(ctx context.Context) (runtime.Result, error) {
		return runtime.Result{}, nil
	}, context.Background())

	_, _, finished := m.FinalResult()
	if finished {
		t.Error("finished = true before execution completed, want false")
	}
}

func TestModel_SignalCancellationQuitsProgram(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	exec := func(c context.Context) (runtime.Result, error) {
		<-c.Done()
		return runtime.Result{}, context.Canceled
	}
	m := New(exec, ctx)

	go cancel()
	msg := m.startProcessing()

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("command = nil, want tea.Quit on cancelled context")
	}
	if reflect.ValueOf(cmd).Pointer() != reflect.ValueOf(tea.Quit).Pointer() {
		t.Errorf("command is not tea.Quit")
	}
}
