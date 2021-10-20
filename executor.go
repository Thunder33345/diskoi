package diskoi

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"sync"
)

type executable interface {
	executor(d discordgo.ApplicationCommandInteractionData) (
		executor *Executor,
		options []*discordgo.ApplicationCommandInteractionDataOption,
		err error,
	)
	applicationCommand() *discordgo.ApplicationCommand
}

//Executor stores the function, the type and parsed information
//add ability to decode input into struct
type Executor struct {
	name        string
	description string
	//fn is the callback for when this slash command is called
	fn interface{}
	//noBindings for slim commands that only accept 2 args
	noBindings bool
	//ty is the type to provide
	ty reflect.Type
	//bindings stores processed information about the ty and also external settings
	bindings []*commandBinding
	m        sync.Mutex
}

var _ executable = (*Executor)(nil)

func NewExecutor(name string, description string, fn interface{}) (*Executor, error) {
	e := Executor{
		name:        name,
		description: description,
		fn:          fn,
	}

	valOf := reflect.ValueOf(fn)
	if valOf.Kind() != reflect.Func {
		return nil, errors.New(fmt.Sprintf("given interface %s(%s) is not type of func", valOf.Type().Name(), valOf.Kind().String()))
	}

	if valOf.Type().NumOut() != 0 {
		return nil, errors.New(fmt.Sprintf("given function(%s) has %d outputs, expecting 0", signature(fn), valOf.Type().NumOut()))
	}

	if valOf.Type().NumIn() < 2 || valOf.Type().NumIn() > 3 {
		return nil, errors.New(fmt.Sprintf("given function(%s) has %d inputs, expecting 2 or 3", signature(fn), valOf.Type().NumIn()))
	}

	if valOf.Type().In(0) != reflect.TypeOf((*discordgo.Session)(nil)) ||
		valOf.Type().In(1) != reflect.TypeOf((*discordgo.InteractionCreate)(nil)) {
		return nil, errors.New(fmt.Sprintf("given function(%s) has incorrect type, expecting func(s *discordgo.Session, i *discordgo.InteractionCreate, ...)", signature(fn)))
	}

	if valOf.Type().NumIn() == 2 {
		e.noBindings = true
		return &e, nil
	}

	if valOf.Type().In(2).Kind() != reflect.Struct {
		return nil, errors.New(fmt.Sprintf("given function(%s) has incorrect type, expecting the 3rd type to be struct not %s", signature(fn), valOf.Type().In(2).Kind().String()))
	}
	e.ty = valOf.Type().In(2)
	var err error
	e.bindings, err = generateBindings(e.ty)
	if err != nil {
		return nil, err
	}
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

func (e *Executor) As(name string, description string) *Executor {
	return &Executor{
		name:        name,
		description: description,
		fn:          e.fn,
		ty:          e.ty,
		bindings:    e.bindings,
	}
}

func (e *Executor) SetChoices(field string, choices []*discordgo.ApplicationCommandOptionChoice) {
	e.m.Lock()
	defer e.m.Unlock()
	for _, b := range e.bindings {
		if b.FieldName == field {
			b.Choices = choices
			return
		}
	}
	r := reflect.TypeOf(e.ty)
	panic(fmt.Sprintf("Failed to set choices: error finding field '%s' on %s(%s)", field, r.Name(), r.Kind()))
}

func (e *Executor) SetChannelTypes(field string, channels []discordgo.ChannelType) {
	e.m.Lock()
	defer e.m.Unlock()
	for _, b := range e.bindings {
		if b.FieldName == field {
			b.ChannelTypes = channels
			return
		}
	}
	r := reflect.TypeOf(e.ty)
	panic(fmt.Sprintf("Failed to set choices: error finding field '%s' on %s(%s)", field, r.Name(), r.Kind()))
}

func (e *Executor) executor(d discordgo.ApplicationCommandInteractionData) (
	executor *Executor,
	options []*discordgo.ApplicationCommandInteractionDataOption,
	err error,
) {
	return e, d.Options, nil
}

func (e *Executor) Execute(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	o []*discordgo.ApplicationCommandInteractionDataOption,
) error {
	f := reflect.ValueOf(e.fn)

	if e.noBindings {
		f.Call([]reflect.Value{reflect.ValueOf(s), reflect.ValueOf(i)})
		return nil
	}

	v, err := generateExecutorValue(s, o, i.GuildID, e)
	if err != nil {
		return err
	}
	f.Call([]reflect.Value{reflect.ValueOf(s), reflect.ValueOf(i), v})
	return nil
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
	o := make([]*discordgo.ApplicationCommandOption, 0, len(e.bindings))
	for _, b := range e.bindings {
		o = append(o, &discordgo.ApplicationCommandOption{
			Type:         b.Type,
			Name:         b.Name,
			Description:  b.Description,
			Required:     b.Required,
			Choices:      b.Choices,
			ChannelTypes: b.ChannelTypes,
		})
	}
	return o
}
