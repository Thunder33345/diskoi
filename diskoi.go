package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"strings"
	"sync"
)

var magicTag = "diskoi"

type Diskoi struct {
	//todo better registration and unregistration and removal func
	//todo some optional error handling func
	//todo config to set panicness
	s                 *discordgo.Session
	remover           func()
	commands          []Executable
	commandsGuild     map[string][]Executable
	registeredCommand map[string]Executable
	m                 sync.Mutex
}

func NewDiskoi() *Diskoi {
	return &Diskoi{
		commandsGuild:     map[string][]Executable{},
		registeredCommand: map[string]Executable{},
		m:                 sync.Mutex{},
	}
}

func (d *Diskoi) RegisterSession(s *discordgo.Session) {
	d.m.Lock()
	defer d.m.Unlock()
	d.s = s
	d.remover = s.AddHandler(d.handle)
}

func (d *Diskoi) RegisterCommands() error {
	d.m.Lock()
	defer d.m.Unlock()
	s := d.s
	for _, cmd := range d.commands {
		cc, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd.applicationCommand())
		if err != nil {
			return err
		}
		d.registeredCommand[cc.ID] = cmd
	}
	for gid, cms := range d.commandsGuild {
		for _, cmd := range cms {
			cc, err := s.ApplicationCommandCreate(s.State.User.ID, gid, cmd.applicationCommand())
			if err != nil {
				return err
			}
			d.registeredCommand[cc.ID] = cmd
		}
	}
	return nil
}

func (d *Diskoi) UnregisterCommands() error {
	d.m.Lock()
	defer d.m.Unlock()
	s := d.s
	for id := range d.registeredCommand {
		err := s.ApplicationCommandDelete(s.State.User.ID, "735145518220443680", id)
		if err != nil {
			return err
		}
		delete(d.registeredCommand, id)
	}
	return nil
}

func (d *Diskoi) AddCommand(exec *Executor) {
	d.m.Lock()
	defer d.m.Unlock()
	d.commands = append(d.commands, exec)
}

func (d *Diskoi) AddGuildCommand(guild string, exec *Executor) {
	d.m.Lock()
	defer d.m.Unlock()
	d.commandsGuild[guild] = append(d.commands, exec)
}

func (d *Diskoi) AddCommandGroup(guild string, name string, description string, cg *CommandGroup) {
	d.m.Lock()
	defer d.m.Unlock()
	if guild == "" {
		d.commands = append(d.commands, &CommandGroupHolder{
			Name:        name,
			Description: description,
			g:           cg,
		})
	} else {
		d.commandsGuild[guild] = append(d.commands, &CommandGroupHolder{
			Name:        name,
			Description: description,
			g:           cg,
		})
	}
}
func (d *Diskoi) handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	id := i.Data.(discordgo.ApplicationCommandInteractionData)
	e, ok := d.registeredCommand[id.ID]
	if !ok {
		return
	}
	//todo proper goroutine queue
	e.Execute(s, i)
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

func splitTag(tag string) map[string]string {
	split := strings.Split(tag, ",")
	res := make(map[string]string, len(split))
	for _, sub := range split {
		kv := strings.SplitN(sub, ":", 2)
		switch len(kv) {
		default:
			continue
		case 1:
			res[kv[0]] = ""
		case 2:
			res[kv[0]] = kv[1]
		}
	}
	return res
}
