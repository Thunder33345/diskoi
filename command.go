package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"reflect"
	"sync"
)

type CommandGroupHolder struct { //todo consider viability of not exporting holders
	Name        string
	Description string
	g           *CommandGroup
	m           sync.Mutex
}

func (c *CommandGroupHolder) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.m.Lock()
	defer c.m.Unlock()
	d := i.Data.(discordgo.ApplicationCommandInteractionData)
	target := d.Options[0]

	grp, in := c.g.findGroup(target.Name)
	if in >= 0 {
		target = target.Options[0]
		sub, _ := grp.Command.findSub(target.Name)
		if sub != nil {
			f := reflect.ValueOf(sub.fn)
			f.Call([]reflect.Value{reflect.ValueOf(s), reflect.ValueOf(i), generateExecutorValue(s, target.Options, i.GuildID, sub)})
			return
		}
	}

	sub, _ := c.g.SubcommandGroup.findSub(target.Name)
	if sub != nil {
		f := reflect.ValueOf(sub.fn)
		f.Call([]reflect.Value{reflect.ValueOf(s), reflect.ValueOf(i), generateExecutorValue(s, target.Options, i.GuildID, sub)})
		return
	}

	panic("TODO: handle impossible case") //todo
}

func (c *CommandGroupHolder) applicationCommand() *discordgo.ApplicationCommand { //todo test
	c.m.Lock()
	defer c.m.Unlock()
	a := &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        c.Name,
		Description: c.Description,
		Options:     []*discordgo.ApplicationCommandOption{},
	}
	a.Options = append(a.Options, c.g.SubcommandGroup.applicationCommandOptions()...)

	for _, s := range c.g.subcommandGroups {
		a.Options = append(a.Options, s.applicationCommandOption())
	}

	return a
}

type CommandGroup struct {
	subcommandGroups []SubcommandGroupHolder
	SubcommandGroup
	m sync.Mutex
}

func NewCommandGroup() *CommandGroup {
	return &CommandGroup{}
}

func (c *CommandGroup) FindSubcommandGroup(name string) (SubcommandGroupHolder, bool) {
	c.m.Lock()
	defer c.m.Unlock()
	h, i := c.findGroup(name)
	if i < 0 {
		return SubcommandGroupHolder{}, false
	}
	return h, true
}

func (c *CommandGroup) AddSubcommandGroup(name string, description string, group *SubcommandGroup) {
	c.m.Lock()
	defer c.m.Unlock()
	_, i := c.findGroup(name)
	h := SubcommandGroupHolder{
		Name:        name,
		Description: description,
		Command:     group,
	}
	if i < 0 {
		c.subcommandGroups = append(c.subcommandGroups, h)
		return
	}
	c.subcommandGroups[i] = h
}

func (c *CommandGroup) RemoveSubcommandGroup(name string) {
	c.m.Lock()
	defer c.m.Unlock()
	_, i := c.findGroup(name)
	if i < 0 {
		return
	}
	c.subcommandGroups = append(c.subcommandGroups[:i], c.subcommandGroups[i+1:]...)
}

func (c *CommandGroup) findGroup(name string) (SubcommandGroupHolder, int) {
	for i, h := range c.subcommandGroups {
		if h.Name == name {
			return h, i
		}
	}
	return SubcommandGroupHolder{}, -1
}

type SubcommandGroupHolder struct {
	Name        string
	Description string
	Command     *SubcommandGroup
}

func (c *SubcommandGroupHolder) applicationCommandOption() *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
		Name:        c.Name,
		Description: c.Description,
		Options:     c.Command.applicationCommandOptions(),
	}
}

type SubcommandGroup struct {
	h []*Executor
	m sync.Mutex
}

func NewSubcommandGroup() *SubcommandGroup {
	return &SubcommandGroup{}
}

func (c *SubcommandGroup) applicationCommandOptions() []*discordgo.ApplicationCommandOption {
	c.m.Lock()
	defer c.m.Unlock()
	o := make([]*discordgo.ApplicationCommandOption, 0, len(c.h))
	for _, e := range c.h {
		o = append(o, &discordgo.ApplicationCommandOption{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        e.name,
			Description: e.description,
			Options:     e.applicationCommandOptions(),
		})
	}
	return o
}
func (c *SubcommandGroup) FindSubcommand(name string) (*Executor, bool) {
	c.m.Lock()
	defer c.m.Unlock()
	h, i := c.findSub(name)
	if i < 0 {
		return nil, false
	}
	return h, true
}

func (c *SubcommandGroup) AddSubcommand(executor *Executor) {
	c.m.Lock()
	defer c.m.Unlock()
	_, i := c.findSub(executor.name)
	if i < 0 {
		c.h = append(c.h, executor)
		return
	}
	c.h[i] = executor
}

func (c *SubcommandGroup) RemoveSubcommand(name string) {
	c.m.Lock()
	defer c.m.Unlock()
	_, i := c.findSub(name)
	if i < 0 {
		return
	}
	c.h = append(c.h[:i], c.h[i+1:]...)
}

func (c *SubcommandGroup) findSub(name string) (*Executor, int) {
	for i, h := range c.h {
		if h.name == name {
			return h, i
		}
	}
	return nil, -1
}

type applicationCommandLister interface {
	applicationCommand() *discordgo.ApplicationCommand
}

type applicationCommandOptionsLister interface {
	applicationCommandOptions() []*discordgo.ApplicationCommandOption
}
