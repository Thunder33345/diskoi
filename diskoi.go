package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"golang.org/x/net/context"
	"sync"
)

type Diskoi struct {
	//idea maybe syncHandling option for go execute
	//todo middlewares for executor command and command group
	s                 *discordgo.Session
	remover           func()
	commands          []Command
	commandsGuild     map[string][]Command
	registeredCommand map[string]Command
	m                 sync.Mutex
	errorHandler      errorHandler
	rawHandler        rawInteractionHandler

	chain MiddlewareChain
}

func NewDiskoi() *Diskoi {
	return &Diskoi{
		commandsGuild:     map[string][]Command{},
		registeredCommand: map[string]Command{},
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
		executor, chain, options, path, err := e.executor(id)
		if err != nil {
			d.getErrorHandler()(s, i, e, CommandParsingError{err: err})
			return
		}
		err = d.Chain().Extend(chain).Then(executor.middleware())(Request{
			ctx:  context.Background(),
			ses:  s,
			ic:   i,
			opts: options,
			meta: &MetaArgument{path: path},
			exec: executor,
		})

		if err != nil {
			d.getErrorHandler()(s, i, e, CommandExecutionError{name: executor.name, err: err})
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
		executor, _, options, path, err := e.executor(id)
		if err != nil {
			d.getErrorHandler()(s, i, e, CommandParsingError{err: err})
			return
		}
		opts, err := executor.autocomplete(s, i, options, &MetaArgument{path: path})
		if err != nil {
			d.getErrorHandler()(s, i, e, AutocompleteExecutionError{name: executor.name, err: err})
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

func (d *Diskoi) SetChain(middlewareChain MiddlewareChain) {
	d.m.Lock()
	defer d.m.Unlock()
	d.chain = middlewareChain
}

func (d *Diskoi) Chain() MiddlewareChain {
	d.m.Lock()
	defer d.m.Unlock()
	return d.chain
}

func (d *Diskoi) findRegisteredCmdById(id string) Command {
	d.m.Lock()
	defer d.m.Unlock()
	cmd, _ := d.registeredCommand[id]
	return cmd
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
