package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"reflect"
	"sync"
)

type CommandGroup struct {
	name             string
	description      string
	subcommandGroups []*SubcommandGroup
	*SubcommandGroup
	m sync.RWMutex
}

func NewCommandGroup(name string, description string) *CommandGroup {
	return &CommandGroup{
		name:            name,
		description:     description,
		SubcommandGroup: &SubcommandGroup{},
	}
}

func (c *CommandGroup) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c.m.RLock()
	defer c.m.RUnlock()
	d, ok := i.Data.(discordgo.ApplicationCommandInteractionData)
	if ok {
		//todo unguarded type assert
	}
	target := d.Options[0]
	sg := c.SubcommandGroup

	grp, in := c.findGroup(target.Name)
	if in >= 0 {
		target = target.Options[0]
		sg = grp
	}

	sg.m.RLock()
	defer sg.m.RUnlock()
	sub, _ := sg.findSub(target.Name)
	if sub != nil {
		f := reflect.ValueOf(sub.fn)
		f.Call([]reflect.Value{reflect.ValueOf(s), reflect.ValueOf(i), generateExecutorValue(s, target.Options, i.GuildID, sub)})
		return
	}

	panic("TODO: handle impossible case") //todo
}

func (c *CommandGroup) applicationCommand() *discordgo.ApplicationCommand { //todo test
	c.m.RLock()
	defer c.m.RUnlock()
	a := &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        c.name,
		Description: c.description,
		Options:     []*discordgo.ApplicationCommandOption{},
	}
	a.Options = append(a.Options, c.SubcommandGroup.applicationCommandOptions()...)

	for _, s := range c.subcommandGroups {
		a.Options = append(a.Options, s.applicationCommandOption())
	}
	return a
}

func (c *CommandGroup) FindSubcommandGroup(name string) (*SubcommandGroup, bool) {
	c.m.RLock()
	defer c.m.RUnlock()
	h, i := c.findGroup(name)
	if i < 0 {
		return nil, false
	}
	return h, true
}

func (c *CommandGroup) AddSubcommandGroup(group *SubcommandGroup) {
	c.m.Lock()
	defer c.m.Unlock()
	_, i := c.findGroup(group.name)
	if i < 0 {
		c.subcommandGroups = append(c.subcommandGroups, group)
		return
	}
	c.subcommandGroups[i] = group
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

func (c *CommandGroup) findGroup(name string) (*SubcommandGroup, int) {
	for i, h := range c.subcommandGroups {
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

type applicationCommandOptionLister interface {
	applicationCommandOption() *discordgo.ApplicationCommandOption
}
