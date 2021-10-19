package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"sync"
)

const magicTag = "diskoi"

type Diskoi struct {
	//todo better registration and unregistration and removal func
	//todo some optional error handling func for command handling
	//todo missing/unregistered command handler
	s                 *discordgo.Session
	remover           func()
	commands          []executable
	commandsGuild     map[string][]executable
	registeredCommand map[string]executable
	m                 sync.Mutex
}

func NewDiskoi() *Diskoi {
	return &Diskoi{
		commandsGuild:     map[string][]executable{},
		registeredCommand: map[string]executable{},
		m:                 sync.Mutex{},
	}
}

func (d *Diskoi) RegisterSession(s *discordgo.Session) {
	d.m.Lock()
	defer d.m.Unlock()
	d.s = s
	d.remover = s.AddHandler(d.handle)
}

func (d *Diskoi) handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand ||
		i.Data.Type() != discordgo.InteractionApplicationCommand {
		return
	}
	id, ok := i.Data.(discordgo.ApplicationCommandInteractionData)
	if !ok {
		return
	}
	e, ok := d.registeredCommand[id.ID]
	if !ok {
		return
	}
	executor, options, err := e.executor(id)
	if err != nil {
		return
	}
	_ = executor.Execute(s, i, options)
}

func (d *Diskoi) Close() error {
	d.m.Lock()
	defer d.m.Unlock()
	d.remover()
	d.commands = nil
	d.commandsGuild = nil
	d.registeredCommand = nil
	d.s = nil
	return nil
}
