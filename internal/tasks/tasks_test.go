package tasks

import (
	"context"
	"testing"

	"github.com/bmc-toolbox/bmclib/v2"
	"github.com/metal-toolbox/bioscfg/internal/model"
	"github.com/metal-toolbox/rivets/condition"
	"github.com/stretchr/testify/assert"
)

type fakeStep struct{}

func (s *fakeStep) Name() string {
	return "fake step"
}

func (s *fakeStep) Run(_ context.Context, _ *bmclib.Client, _ sharedData) (string, error) {
	return "", nil
}

type fakeTask struct {
	asset *model.Asset
	steps []Step
}

func newFakeTask() *fakeTask {
	return &fakeTask{
		asset: &model.Asset{},
		steps: []Step{&fakeStep{}},
	}
}

func newFakeRCTask() *RCTask {
	return &RCTask{}
}

func (t *fakeTask) Name() string {
	return "fake task"
}

func (t *fakeTask) Asset() *model.Asset {
	return t.asset
}

func (t *fakeTask) Steps() []Step {
	return t.steps
}

type fakePublisher struct {
	t *testing.T
}

func (m *fakePublisher) Publish(_ context.Context, _ *condition.Task[any, any], _ bool) error {
	return nil
}

func TestTaskRunnerHandlePanic(t *testing.T) {
	task := newFakeTask()
	rcTask := newFakeRCTask()
	runner := NewTaskRunner(&fakePublisher{t: t}, task, rcTask)

	err := runner.Run(context.Background(), nil)

	if assert.NotNil(t, err) {
		assert.Equal(t, "Task fatal error, check logs for details", err.Error())
	}
}
