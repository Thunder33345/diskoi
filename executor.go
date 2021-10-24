package diskoi

import (
	"diskoi/parser"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"sync"
)

//Executor stores the function, the type and parsed information
//add ability to decode input into struct
type Executor struct { //todo rearrange all methods
	name        string
	description string
	data        *parser.Data
	m           sync.Mutex
}

var _ Command = (*Executor)(nil)

func NewExecutor(name string, description string, fn interface{}) (*Executor, error) {
	e := Executor{
		name:        name,
		description: description,
	}
	data, err := parser.Analyze(fn)
	if err != nil {
		return nil, err
	}
	e.data = data
	return &e, nil
}

func MustNewExecutor(name string, description string, fn interface{}) *Executor {
	executor, err := NewExecutor(name, description, fn)
	if err != nil {
		panic(err)
	}
	return executor
}

func (e *Executor) Name() string {
	return e.name
}

func (e *Executor) Description() string {
	return e.description
}

func (e *Executor) ArgumentByName(name string) *parser.PayloadArgument {
	return e.data.ArgumentByName(name)
}

func (e *Executor) ArgumentByIndex(index []int) *parser.PayloadArgument {
	return e.data.ArgumentByIndex(index)
}

func (e *Executor) As(name string, description string) *Executor {
	return &Executor{
		name:        name,
		description: description,
		data:        e.data,
	}
}

func (e *Executor) Lock() {
	e.m.Lock()
}

func (e *Executor) Unlock() {
	e.m.Unlock()
}
func (e *Executor) executor(d discordgo.ApplicationCommandInteractionData) (
	executor *Executor,
	options []*discordgo.ApplicationCommandInteractionDataOption,
	err error,
) {
	return e, d.Options, nil
}

func (e *Executor) execute(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	o []*discordgo.ApplicationCommandInteractionDataOption,
) error {
	err := e.data.Execute(s, i, o, parser.DiskoiData{}) //todo fill in diskoi data
	if err != nil {
		return errors.New(fmt.Sprintf("error running command %s: %v", e.name, err))
	}
	return nil
}

func (e *Executor) applicationCommand() *discordgo.ApplicationCommand {
	e.m.Lock()
	defer e.m.Unlock()
	return &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        e.name,
		Description: e.description,
		Options:     e.data.ApplicationCommandOptions(),
	}
}

func (e *Executor) applicationCommandOptions() []*discordgo.ApplicationCommandOption {
	e.m.Lock()
	defer e.m.Unlock()
	return e.data.ApplicationCommandOptions()
}
