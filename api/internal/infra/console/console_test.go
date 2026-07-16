package console

import "testing"

type testCommand struct{}

func (testCommand) Name() string        { return "test" }
func (testCommand) Description() string { return "test command" }
func (testCommand) Usage() string       { return "test" }
func (testCommand) Run([]string) error  { return nil }

type structuredOutputCommand struct{ testCommand }

func (structuredOutputCommand) SuppressCompletionOutput() bool { return true }

func TestShouldPrintCompletionOutput(t *testing.T) {
	if !shouldPrintCompletionOutput(testCommand{}) {
		t.Fatal("plain command completion output = false, want true")
	}
	if shouldPrintCompletionOutput(structuredOutputCommand{}) {
		t.Fatal("structured command completion output = true, want false")
	}
}
