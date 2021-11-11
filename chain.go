package diskoi

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
