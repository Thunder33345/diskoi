package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"golang.org/x/net/context"
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

type Middleware func(r Request) error
type Chainer func(next Middleware) Middleware

type Chain struct {
	builders []Chainer
}

func NewChain(builders ...Chainer) Chain {
	return Chain{builders: builders}
}

func (c Chain) Then(m Middleware) Middleware {
	for i := len(c.builders) - 1; i >= 0; i-- {
		m = c.builders[i](m)
	}
	return m
}

func (c Chain) Append(builders ...Chainer) Chain {
	nc := make([]Chainer, 0, len(c.builders)+len(builders))
	nc = append(nc, c.builders...)
	nc = append(nc, builders...)
	return Chain{builders: nc}
}

func (c Chain) Extend(chain Chain) Chain {
	return c.Append(chain.builders...)
}
