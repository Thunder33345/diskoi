package diskoi

import (
	"context"
	"github.com/bwmarrin/discordgo"
)

type Request struct {
	ctx  context.Context
	ses  *discordgo.Session
	ic   *discordgo.InteractionCreate
	opts []*discordgo.ApplicationCommandInteractionDataOption
	meta *MetaArgument
	exec *Executor
}

func (c *Request) Context() context.Context {
	return c.ctx
}

func (c *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		panic("nil context")
	}
	cc := new(Request)
	*cc = *c
	cc.ctx = ctx
	return cc
}

func (c *Request) Session() *discordgo.Session {
	return c.ses
}

func (c *Request) Interaction() *discordgo.InteractionCreate {
	return c.ic
}

func (c *Request) Options() []*discordgo.ApplicationCommandInteractionDataOption {
	return c.opts
}

func (c *Request) Meta() *MetaArgument {
	return c.meta
}

func (c *Request) Executor() *Executor {
	return c.exec
}
