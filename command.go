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

	chain Chain
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

func (c *CommandGroup) SetChain(chain Chain) {
	c.m.Lock()
	defer c.m.Unlock()
	c.chain = chain
}

func (c *CommandGroup) Chain() Chain {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.chain
}

func (c *CommandGroup) execute(s *discordgo.Session, i *discordgo.InteractionCreate, pre Chain) error {
	id, ok := i.Data.(discordgo.ApplicationCommandInteractionData)
	if !ok {
		return newDiscordExpectationError(
			fmt.Sprintf(`given interaction data is not ApplicationCommandInteractionData in command group "%s"`, c.name))
	}
	exec, grpChain, opts, meta, err := c.findExecutor(id)
	if err != nil {
		return err
	}
	chain := pre.Extend(grpChain)
	err = exec.executeWithOpts(s, i, chain, opts, meta)
	if err != nil {
		return err
	}
	return nil
}

func (c *CommandGroup) autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) ([]*discordgo.ApplicationCommandOptionChoice, error) {
	id, ok := i.Data.(discordgo.ApplicationCommandInteractionData)
	if !ok {
		return nil, newDiscordExpectationError(
			fmt.Sprintf(`given interaction data is not ApplicationCommandInteractionData in command group "%s"`, c.name))
	}
	exec, _, opts, meta, err := c.findExecutor(id)
	if err != nil {
		return nil, err
	}
	return exec.autocompleteWithOps(s, i, opts, meta)
}

func (c *CommandGroup) findExecutor(d discordgo.ApplicationCommandInteractionData) (
	*Executor, Chain, []*discordgo.ApplicationCommandInteractionDataOption, *MetaArgument, error,
) {
	c.m.RLock()
	defer c.m.RUnlock()
	path := make([]string, 0, 3)
	path = append(path, c.name)
	if len(d.Options) <= 0 {
		return nil, Chain{}, nil, nil, newDiscordExpectationError("missing options: expecting options given for command group, none given for" + errPath(path))
	}
	target := d.Options[0]
	chain := c.chain

	var group *SubcommandGroup
	switch {
	case target.Type == discordgo.ApplicationCommandOptionSubCommand:
		group = c.SubcommandGroup
	case target.Type == discordgo.ApplicationCommandOptionSubCommandGroup:
		group, _ = c.findGroup(target.Name)
		path = append(path, target.Name)
		if group == nil {
			return nil, Chain{}, nil, nil, CommandParsingError{err: fmt.Errorf(`missing subcommand group: group "%s" not found on %s`, target.Name, errPath(path))}
		}
		//if so we unwrap options to get the actual name
		chain = chain.Extend(group.Chain())
		target = target.Options[0]
	default:
		return nil, Chain{}, nil, nil, newDiscordExpectationError(fmt.Sprintf(
			`non command option type: expecting "SubCommand" or "SubCommandGroup" command option type but received "%s" for %s`,
			target.Type.String(), errPath(path)))
	}

	var sub *Executor
	withRWMutex(&group.m, func() {
		sub, _ = group.findSub(target.Name)
	})
	path = append(path, target.Name)
	if sub != nil {
		return sub, chain, target.Options, &MetaArgument{path: path}, nil
	}
	return nil, Chain{}, nil, nil, CommandParsingError{err: fmt.Errorf(`missing subcommand: subcommand "%s" not found on %s`, target.Name, errPath(path))}
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

func (c *CommandGroup) lock() {
	c.m.RLock()
	defer c.m.RUnlock()
	c.SubcommandGroup.lock()
	for _, grp := range c.subcommandGroups {
		grp.lock()
	}
}
