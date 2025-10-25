package commands

import (
	"testing"
)

func TestNewGenerateCommand(t *testing.T) {
	cmd := NewGenerateCommand()

	if cmd.Use != "generate" {
		t.Errorf("expected Use to be 'generate', got %s", cmd.Use)
	}

	// Check aliases
	found := false
	for _, alias := range cmd.Aliases {
		if alias == "g" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected alias 'g' to be registered")
	}

	// Check subcommands are registered
	expectedSubcommands := []string{
		"resource",
		"controller",
		"migration",
	}

	for _, expected := range expectedSubcommands {
		found := false
		for _, cmd := range cmd.Commands() {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %s to be registered", expected)
		}
	}
}
