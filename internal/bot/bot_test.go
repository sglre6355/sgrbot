package bot

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestNewBot(t *testing.T) {
	cfg := &Config{
		DiscordToken: "test-token",
	}

	b := NewBot(cfg)

	if b == nil {
		t.Fatal("expected bot to be created, got nil")
	}
	if b.config != cfg {
		t.Error("expected config to be stored")
	}
}

func TestBot_LoadModules_InitializesModules(t *testing.T) {
	cfg := &Config{DiscordToken: "test-token"}
	b := NewBot(cfg)

	initCalled := false
	mod := &stubModule{
		name:    "test",
		initErr: nil,
	}
	// Track if Init was called by wrapping
	origMod := mod
	b.modules = []Module{origMod}

	// Use a custom stub that tracks init
	trackingMod := &trackingStubModule{
		stubModule: stubModule{name: "tracking"},
		initCalled: &initCalled,
	}
	b.modules = []Module{trackingMod}

	err := b.initModules()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !initCalled {
		t.Error("expected Init to be called")
	}
}

func TestBot_LoadModules_ReturnsInitError(t *testing.T) {
	cfg := &Config{DiscordToken: "test-token"}
	b := NewBot(cfg)

	expectedErr := errors.New("init failed")
	mod := &stubModule{
		name:    "failing",
		initErr: expectedErr,
	}
	b.modules = []Module{mod}

	err := b.initModules()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestBot_BuildHandlerMap(t *testing.T) {
	cfg := &Config{DiscordToken: "test-token"}
	b := NewBot(cfg)

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, r Responder) error {
		return nil
	}

	mod := &stubModule{
		name: "test",
		handlers: map[string]InteractionHandler{
			"ping": handler,
		},
	}
	b.modules = []Module{mod}

	b.buildHandlerMap()

	if _, ok := b.handlers["ping"]; !ok {
		t.Error("expected ping handler to be registered")
	}
}

func TestBot_BuildHandlerMap_MultipleModules(t *testing.T) {
	cfg := &Config{DiscordToken: "test-token"}
	b := NewBot(cfg)

	handler1 := func(s *discordgo.Session, i *discordgo.InteractionCreate, r Responder) error {
		return nil
	}
	handler2 := func(s *discordgo.Session, i *discordgo.InteractionCreate, r Responder) error {
		return nil
	}

	mod1 := &stubModule{
		name: "mod1",
		handlers: map[string]InteractionHandler{
			"cmd1": handler1,
		},
	}
	mod2 := &stubModule{
		name: "mod2",
		handlers: map[string]InteractionHandler{
			"cmd2": handler2,
		},
	}
	b.modules = []Module{mod1, mod2}

	b.buildHandlerMap()

	if len(b.handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(b.handlers))
	}
}

func TestBot_CollectCommands(t *testing.T) {
	cfg := &Config{DiscordToken: "test-token"}
	b := NewBot(cfg)

	cmd := &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Ping command",
	}

	mod := &stubModule{
		name:     "test",
		commands: []*discordgo.ApplicationCommand{cmd},
	}
	b.modules = []Module{mod}

	commands := b.collectCommands()

	if len(commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(commands))
	}
	if commands[0].Name != "ping" {
		t.Errorf("expected command name %q, got %q", "ping", commands[0].Name)
	}
}

// trackingStubModule is a stub that tracks if Init was called
type trackingStubModule struct {
	stubModule
	initCalled *bool
}

func (m *trackingStubModule) Init(deps ModuleDependencies) error {
	*m.initCalled = true
	return m.stubModule.Init(deps)
}
