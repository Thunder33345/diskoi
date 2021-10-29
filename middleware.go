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
type MiddlewareBuilder func(next Middleware) Middleware

type MiddlewareChain struct {
	builders []MiddlewareBuilder
}

func NewMiddlewareChain(builders ...MiddlewareBuilder) MiddlewareChain {
	return MiddlewareChain{builders: builders}
}

func (c MiddlewareChain) Then(m Middleware) Middleware {
	for i := len(c.builders) - 1; i >= 0; i-- {
		m = c.builders[i](m)
	}
	return m
}

func (c MiddlewareChain) Append(builders ...MiddlewareBuilder) MiddlewareChain {
	nc := make([]MiddlewareBuilder, 0, len(c.builders)+len(builders))
	nc = append(nc, c.builders...)
	nc = append(nc, builders...)
	return MiddlewareChain{builders: nc}
}

func (c MiddlewareChain) Extend(chain MiddlewareChain) MiddlewareChain {
	return c.Append(chain.builders...)
}
