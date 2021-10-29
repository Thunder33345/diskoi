package diskoi

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"sync"
)

type CommandGroup struct {
	name             string
	description      string
	subcommandGroups []*SubcommandGroup
	*SubcommandGroup
	m sync.RWMutex

	middlewareChain MiddlewareChain
}

var _ Command = (*CommandGroup)(nil)

func NewCommandGroup(name string, description string) *CommandGroup {
	return &CommandGroup{
		name:            name,
		description:     description,
		SubcommandGroup: &SubcommandGroup{},
	}
}

func (c *CommandGroup) Name() string {
	return c.name
}

func (c *CommandGroup) Description() string {
	return c.description
}

func (c *CommandGroup) SetChain(middlewareChain MiddlewareChain) {
	c.m.Lock()
	defer c.m.Unlock()
	c.middlewareChain = middlewareChain
}

func (c *CommandGroup) Chain() MiddlewareChain {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.middlewareChain
}

func (c *CommandGroup) executor(d discordgo.ApplicationCommandInteractionData) (
	*Executor, MiddlewareChain, []*discordgo.ApplicationCommandInteractionDataOption, []string, error,
) {
	c.m.RLock()
	defer c.m.RUnlock()
	path := make([]string, 0, 3)
	path = append(path, c.name)
	if len(d.Options) <= 0 {
		return nil, MiddlewareChain{}, nil, nil, newDiscordExpectationError("missing options: expecting options given for command group, none given for" + errPath(path))
	}
	target := d.Options[0]
	chain := c.middlewareChain

	var group *SubcommandGroup
	switch {
	case target.Type == discordgo.ApplicationCommandOptionSubCommand:
		group = c.SubcommandGroup
	case target.Type == discordgo.ApplicationCommandOptionSubCommandGroup:
		group, _ = c.findGroup(target.Name)
		path = append(path, target.Name)
		if group == nil {
			return nil, MiddlewareChain{}, nil, nil, fmt.Errorf(`missing subcommand group: group "%s" not found on %s`, target.Name, errPath(path))
		}
		//if so we unwrap options to get the actual name
		withMutex(&group.m, func() {
			chain = chain.Extend(group.middlewareChain)
		})
		target = target.Options[0]
	default:
		return nil, MiddlewareChain{}, nil, nil, newDiscordExpectationError(fmt.Sprintf(
			`non command option type: expecting "SubCommand" or "SubCommandGroup" command option type but received "%s" for %s`,
			target.Type.String(), errPath(path)))
	}

	var sub *Executor
	withMutex(&group.m, func() {
		sub, _ = group.findSub(target.Name)
	})
	path = append(path, target.Name)
	if sub != nil {
		return sub, chain, target.Options, path, nil
	}
	return nil, MiddlewareChain{}, nil, nil, fmt.Errorf(`missing subcommand: subcommand "%s" not found on %s`, target.Name, errPath(path))
}

func withMutex(m *sync.RWMutex, f func()) {
	m.RLock()
	defer m.RUnlock()
	f()
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
