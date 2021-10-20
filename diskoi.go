package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"sync"
)

const magicTag = "diskoi"

type Diskoi struct {
	//todo better registration and un-registration and removal func
	//idea maybe syncHandling option for go execute
	s                 *discordgo.Session
	remover           func()
	commands          []executable
	commandsGuild     map[string][]executable
	registeredCommand map[string]executable
	m                 sync.Mutex
	errorHandler      errorHandler
	rawHandler        rawInteractionHandler
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
	e, ok := d.registeredCmd(id.ID)
	if !ok {
		d.getRawHandler()(s, i)
		return
	}
	executor, options, err := e.executor(id)
	if err != nil {
		d.getErrorHandler()(s, i, e, CommandParsingError{err: err})
		return
	}
	err = executor.Execute(s, i, options)
	if err != nil {
		d.getErrorHandler()(s, i, e, CommandExecutionError{err: err})
	}
}

func (d *Diskoi) registeredCmd(id string) (executable, bool) {
	d.m.Lock()
	defer d.m.Unlock()
	e, ok := d.registeredCommand[id]
	return e, ok
}

func (d *Diskoi) SetErrorHandler(handler errorHandler) {
	d.m.Lock()
	defer d.m.Unlock()
	d.errorHandler = handler
}

func (d *Diskoi) getErrorHandler() errorHandler {
	d.m.Lock()
	defer d.m.Unlock()
	return d.errorHandler
}

func (d *Diskoi) SetRawHandler(handler rawInteractionHandler) {
	d.m.Lock()
	defer d.m.Unlock()
	d.rawHandler = handler
}

func (d *Diskoi) getRawHandler() rawInteractionHandler {
	d.m.Lock()
	defer d.m.Unlock()
	return d.rawHandler
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

type errorHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, exec executable, err error)
type rawInteractionHandler func(*discordgo.Session, *discordgo.InteractionCreate)
