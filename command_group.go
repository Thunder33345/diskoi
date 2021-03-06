package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"sync"
)

type SubcommandGroup struct {
	name        string
	description string
	h           []*Executor
	m           sync.RWMutex

	chain Chain
}

func NewSubcommandGroup(name string, description string) *SubcommandGroup {
	return &SubcommandGroup{
		name:        name,
		description: description,
	}
}

func (c *SubcommandGroup) SetChain(chain Chain) {
	c.m.Lock()
	defer c.m.Unlock()
	c.chain = chain
}

func (c *SubcommandGroup) Chain() Chain {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.chain
}

func (c *SubcommandGroup) applicationCommandOption() *discordgo.ApplicationCommandOption {
	c.m.RLock()
	defer c.m.RUnlock()
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
		Name:        c.name,
		Description: c.description,
		Options:     c.applicationCommandOptionsUnsafe(),
	}
}

func (c *SubcommandGroup) applicationCommandOptions() []*discordgo.ApplicationCommandOption {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.applicationCommandOptionsUnsafe()
}

func (c *SubcommandGroup) applicationCommandOptionsUnsafe() []*discordgo.ApplicationCommandOption {
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
	c.m.RLock()
	defer c.m.RUnlock()
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

func (c *SubcommandGroup) lock() {
	c.m.Lock()
	defer c.m.Unlock()
	for _, exec := range c.h {
		exec.lock()
	}
}
