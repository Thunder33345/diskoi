package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"sync"
)

type CommandGroup struct {
	//todo anti colision of SubcommandGroup and subcommandGroups
	//todo max element count limit
	name             string
	description      string
	subcommandGroups []*SubcommandGroup
	*SubcommandGroup
	m sync.RWMutex
}

var _ executable = (*CommandGroup)(nil)

func NewCommandGroup(name string, description string) *CommandGroup {
	return &CommandGroup{
		name:            name,
		description:     description,
		SubcommandGroup: &SubcommandGroup{},
	}
}

func (c *CommandGroup) executor(d discordgo.ApplicationCommandInteractionData) (
	executor *Executor,
	options []*discordgo.ApplicationCommandInteractionDataOption,
	err error,
) {
	c.m.RLock()
	defer c.m.RUnlock()
	//initialize default fallbacks
	//target is the name of a subcommand or a group
	target := d.Options[0] //todo handle failures
	//default embedded subcommand group to search
	sg := c.SubcommandGroup

	//find if there's a group named as such
	grp, _ := c.findGroup(target.Name)
	if grp != nil {
		//if so we unwrap options to get the actual name
		target = target.Options[0]
		//and also overwrite the default subcommand to said group
		sg = grp
	}

	sg.m.RLock()
	defer sg.m.RUnlock()
	sub, _ := sg.findSub(target.Name)
	if sub != nil {
		return sub, target.Options, nil
	}
	return nil, nil, MissingSubcommandError{}
}

func (c *CommandGroup) applicationCommand() *discordgo.ApplicationCommand {
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
