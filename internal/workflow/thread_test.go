package workflow

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
)

func TestThreads_AllFinished(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) *threads
		expected bool
	}{
		{
			name: "empty threads map returns true",
			setup: func(_ *testing.T) *threads {
				return newThreads()
			},
			expected: true,
		},
		{
			name: "all threads finished returns true",
			setup: func(_ *testing.T) *threads {
				ts := newThreads()
				th1 := ts.New(0, workflow.NewExecID(0))
				th1.SetState(ThreadFinished)
				th2 := ts.New(1, workflow.NewExecID(1))
				th2.SetState(ThreadFinished)
				return ts
			},
			expected: true,
		},
		{
			name: "one thread still running returns false",
			setup: func(_ *testing.T) *threads {
				ts := newThreads()
				th1 := ts.New(0, workflow.NewExecID(0))
				th1.SetState(ThreadFinished)
				ts.New(1, workflow.NewExecID(1)) // default is ThreadRunning
				return ts
			},
			expected: false,
		},
		{
			name: "single running thread returns false",
			setup: func(_ *testing.T) *threads {
				ts := newThreads()
				ts.New(0, workflow.NewExecID(0))
				return ts
			},
			expected: false,
		},
		{
			name: "single finished thread returns true",
			setup: func(_ *testing.T) *threads {
				ts := newThreads()
				th := ts.New(0, workflow.NewExecID(0))
				th.SetState(ThreadFinished)
				return ts
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ts := tt.setup(t)

			// Act
			result := ts.AllFinished()

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}
