package bot

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

// stubModule is a test double for Module
type stubModule struct {
	name          string
	commands      []*discordgo.ApplicationCommand
	handlers      map[string]InteractionHandler
	eventHandlers []EventHandler
	initErr       error
	shutErr       error
}

func (m *stubModule) Name() string                                   { return m.name }
func (m *stubModule) Commands() []*discordgo.ApplicationCommand      { return m.commands }
func (m *stubModule) CommandHandlers() map[string]InteractionHandler { return m.handlers }
func (m *stubModule) EventHandlers() []EventHandler                  { return m.eventHandlers }
func (m *stubModule) Init(deps ModuleDependencies) error             { return m.initErr }
func (m *stubModule) Shutdown() error                                { return m.shutErr }

func TestRegistry_Register(t *testing.T) {
	// Use a fresh registry for testing
	reg := NewRegistry()

	mod := &stubModule{name: "test-module"}
	reg.Register(mod)

	modules := reg.Modules()
	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(modules))
	}

	if modules[0].Name() != "test-module" {
		t.Errorf("expected module name %q, got %q", "test-module", modules[0].Name())
	}
}

func TestRegistry_RegisterMultiple(t *testing.T) {
	reg := NewRegistry()

	mod1 := &stubModule{name: "module-1"}
	mod2 := &stubModule{name: "module-2"}

	reg.Register(mod1)
	reg.Register(mod2)

	modules := reg.Modules()
	if len(modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(modules))
	}
}

func TestRegistry_ModulesReturnsSnapshot(t *testing.T) {
	reg := NewRegistry()

	mod1 := &stubModule{name: "module-1"}
	reg.Register(mod1)

	modules := reg.Modules()

	// Register another module after getting snapshot
	mod2 := &stubModule{name: "module-2"}
	reg.Register(mod2)

	// Original snapshot should not be affected
	if len(modules) != 1 {
		t.Errorf("expected snapshot to have 1 module, got %d", len(modules))
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Clear global registry before test
	ResetGlobalRegistry()

	mod := &stubModule{name: "global-test"}
	Register(mod)

	modules := Modules()
	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(modules))
	}

	if modules[0].Name() != "global-test" {
		t.Errorf("expected module name %q, got %q", "global-test", modules[0].Name())
	}

	// Clean up
	ResetGlobalRegistry()
}
