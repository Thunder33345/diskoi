package diskoi

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"sync"
)

//Executor stores the function, the type and parsed information
type Executor struct {
	name        string
	description string
	m           sync.Mutex

	//fn is the callback function
	fn interface{}
	//fnArg is a slice of arguments taken by the function
	fnArg []*fnArgument
	//cmdStruct is the struct that will be parsed as the arguments for discord
	//cmdStruct must be the last argument of the fn, if exist
	cmdStruct reflect.Type
	//cmdArg is a slice command arguments from the struct, used for generating ApplicationCommandOptions
	cmdArg []*commandArgument
	//cmdSpecialArg holds slice of special meta arguments
	cmdSpecialArg []*specialArgument
}

var _ Command = (*Executor)(nil)

func NewExecutor(name string, description string, fn interface{}) (*Executor, error) {
	e := Executor{
		name:        name,
		description: description,
	}
	fnArgs, cmdStruct, cmdArg, specArg, err := analyzeCmdFn(fn)
	if err != nil {
		return nil, fmt.Errorf(`failed to parse command "%s": %w`, name, err)
	}
	e.fn, e.fnArg, e.cmdStruct, e.cmdArg, e.cmdSpecialArg = fn, fnArgs, cmdStruct, cmdArg, specArg
	return &e, nil
}

func MustNewExecutor(name string, description string, fn interface{}) *Executor {
	executor, err := NewExecutor(name, description, fn)
	if err != nil {
		panic(fmt.Errorf("error creating executor named %s: %w", name, err))
	}
	return executor
}

func (e *Executor) Name() string {
	return e.name
}

func (e *Executor) Description() string {
	return e.description
}

func (e *Executor) As(name string, description string) *Executor {
	return &Executor{
		name:          name,
		description:   description,
		fn:            e.fn,
		fnArg:         e.fnArg,
		cmdStruct:     e.cmdStruct,
		cmdArg:        e.cmdArg,
		cmdSpecialArg: e.cmdSpecialArg,
	}
}

func (e *Executor) executor(d discordgo.ApplicationCommandInteractionData) (
	*Executor,
	[]*discordgo.ApplicationCommandInteractionDataOption,
	[]string,
	error,
) {
	return e, d.Options, []string{e.name}, nil
}

func (e *Executor) execute(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	opts []*discordgo.ApplicationCommandInteractionDataOption,
	meta *metaArgument,
) error {
	values, err := reconstructFunctionArgs(e.fnArg, e.cmdArg, e.cmdSpecialArg, meta, s, i, opts)
	if err != nil {
		return fmt.Errorf(`error reconstructing command "%s": %w`, e.name, err)
	}
	fn := reflect.ValueOf(e.fn)
	fn.Call(values)
	return nil
}

func (e *Executor) autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate,
	opts []*discordgo.ApplicationCommandInteractionDataOption, meta *metaArgument) ([]*discordgo.ApplicationCommandOptionChoice, error) {
	arg, values, err := reconstructAutocompleteArgs(e.cmdArg, e.cmdSpecialArg, meta, s, i, opts)
	if err != nil {
		return nil, fmt.Errorf(`error autocompleting command "%s": %w`, e.name, err)
	}
	rets := reflect.ValueOf(arg.autocompleteFn).Call(values)
	optChoice := rets[0].Interface().([]*discordgo.ApplicationCommandOptionChoice)
	return optChoice, nil
}

func (e *Executor) applicationCommand() *discordgo.ApplicationCommand {
	e.m.Lock()
	defer e.m.Unlock()
	return &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        e.name,
		Description: e.description,
		Options:     e.applicationCommandOptions(),
	}
}

func (e *Executor) applicationCommandOptions() []*discordgo.ApplicationCommandOption {
	o := make([]*discordgo.ApplicationCommandOption, 0, len(e.cmdArg))
	for _, b := range e.cmdArg {
		o = append(o, &discordgo.ApplicationCommandOption{
			Type:         b.cType,
			Name:         b.Name,
			Description:  b.Description,
			Required:     b.Required,
			Choices:      b.Choices,
			ChannelTypes: b.ChannelTypes,
			Autocomplete: b.autocompleteFn != nil,
		})
	}
	return o
}

func (e *Executor) findField(name string) (*commandArgument, error) {
	for _, arg := range e.cmdArg {
		if arg.fieldName == name {
			return arg, nil
		}
	}
	return nil, fmt.Errorf(`error finding field named "%s" in command "%s"`, name, e.name)
}

func (e *Executor) SetName(fieldName string, name string) error {
	e.m.Lock()
	defer e.m.Unlock()
	arg, err := e.findField(fieldName)
	if err != nil {
		return err
	}
	arg.Name = name
	return nil
}

func (e *Executor) SetDescription(fieldName string, desc string) error {
	e.m.Lock()
	defer e.m.Unlock()
	arg, err := e.findField(fieldName)
	if err != nil {
		return err
	}
	arg.Description = desc
	return nil
}

func (e *Executor) SetRequired(fieldName string, required bool) error {
	e.m.Lock()
	defer e.m.Unlock()
	arg, err := e.findField(fieldName)
	if err != nil {
		return err
	}
	arg.Required = required
	return nil
}

func (e *Executor) SetChoices(fieldName string, choices []*discordgo.ApplicationCommandOptionChoice) error {
	e.m.Lock()
	defer e.m.Unlock()
	arg, err := e.findField(fieldName)
	if err != nil {
		return err
	}
	arg.Choices = choices
	return nil
}

func (e *Executor) SetChannelTypes(fieldName string, ChannelTypes []discordgo.ChannelType) error {
	e.m.Lock()
	defer e.m.Unlock()
	arg, err := e.findField(fieldName)
	if err != nil {
		return err
	}
	arg.ChannelTypes = ChannelTypes
	return nil
}

func (e *Executor) SetAutoComplete(fieldName string, fn interface{}) error {
	e.m.Lock()
	defer e.m.Unlock()
	arg, err := e.findField(fieldName)
	if err != nil {
		return err
	}
	fnArgs, err := analyzeAutocompleteFunction(fn, e.cmdStruct)
	if err != nil {
		return fmt.Errorf(`error analyzing autocomplete function in "%s": %w`, e.name, err)
	}
	arg.autocompleteFn = fn
	arg.autocompleteArgs = fnArgs
	return nil
}
