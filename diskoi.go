package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"sync"
)

type Diskoi struct {
	//idea maybe syncHandling option for go execute
	s                 *discordgo.Session
	remover           func()
	commands          []Command
	commandsGuild     map[string][]Command
	registeredCommand map[string]registerMapping
	m                 sync.Mutex
	errorHandler      errorHandler
	rawHandler        rawInteractionHandler

	chain Chain
}

func NewDiskoi() *Diskoi {
	return &Diskoi{
		commandsGuild:     map[string][]Command{},
		registeredCommand: map[string]registerMapping{},
		m:                 sync.Mutex{},
		errorHandler:      func(s *discordgo.Session, i *discordgo.InteractionCreate, cmd Command, err error) {},
		rawHandler:        func(session *discordgo.Session, create *discordgo.InteractionCreate) {},
	}
}

func (d *Diskoi) RegisterSession(s *discordgo.Session) {
	d.m.Lock()
	defer d.m.Unlock()
	d.s = s
	d.remover = s.AddHandler(d.handle)
}

func (d *Diskoi) handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch {
	case i.Type == discordgo.InteractionApplicationCommand && i.Data.Type() == discordgo.InteractionApplicationCommand:
		id, ok := i.Data.(discordgo.ApplicationCommandInteractionData)
		if !ok {
			return
		}
		e := d.findRegisteredCmdById(id.ID)
		if e == nil {
			d.getRawHandler()(s, i)
			return
		}

		err := e.execute(s, i, d.chain)

		if err != nil {
			d.getErrorHandler()(s, i, e, err)
		}
	case i.Type == discordgo.InteractionApplicationCommandAutocomplete &&
		i.Data.Type() == discordgo.InteractionApplicationCommand:
		id, ok := i.Data.(discordgo.ApplicationCommandInteractionData)
		if !ok {
			return
		}
		e := d.findRegisteredCmdById(id.ID)
		if e == nil {
			d.getRawHandler()(s, i)
			return
		}
		opts, err := e.autocomplete(s, i)

		if err != nil {
			d.getErrorHandler()(s, i, e, AutocompleteExecutionError{name: e.Name(), err: err})
		}
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: opts,
			},
		})
		if err != nil {
			d.getErrorHandler()(s, i, e, DiscordAPIError{err: err})
		}
	}
}

func (d *Diskoi) SetChain(chain Chain) {
	d.m.Lock()
	defer d.m.Unlock()
	d.chain = chain
}

func (d *Diskoi) Chain() Chain {
	d.m.Lock()
	defer d.m.Unlock()
	return d.chain
}

func (d *Diskoi) findRegisteredCmdById(id string) Command {
	d.m.Lock()
	defer d.m.Unlock()
	cmd, _ := d.registeredCommand[id]
	return cmd.command
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
