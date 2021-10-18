package diskoi

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"sync"
)

type Executable interface {
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate)
	applicationCommand() *discordgo.ApplicationCommand
}

//Executor stores the function, the type and parsed information
//add ability to decode input into struct
type Executor struct {
	name        string
	description string
	//fn is the callback for when this slash command is called
	fn interface{}
	//ty is the type to provide
	ty reflect.Type
	//bindings stores processed information about the ty and also external settings
	bindings []*commandBinding
	m        sync.Mutex
}

func NewExecutor(name string, description string, fn interface{}) *Executor {
	e := Executor{
		name:        name,
		description: description,
		fn:          fn,
	}

	valOf := reflect.ValueOf(fn)
	if valOf.Kind() != reflect.Func {
		panic(fmt.Sprintf("given interface %s(%s) is not type of func", valOf.Type().Name(), valOf.Kind().String()))
	}

	if valOf.Type().NumOut() != 0 {
		panic(fmt.Sprintf("given function(%s) has %d outputs, expecting 0", signature(fn), valOf.Type().NumOut()))
	}

	if valOf.Type().NumIn() != 3 {
		panic(fmt.Sprintf("given function(%s) has %d inputs, expecting 3", signature(fn), valOf.Type().NumIn()))
	}

	if valOf.Type().In(0) != reflect.TypeOf((*discordgo.Session)(nil)) ||
		valOf.Type().In(1) != reflect.TypeOf((*discordgo.InteractionCreate)(nil)) {
		panic(fmt.Sprintf("given function(%s) has incorrect type, expecting func(s *discordgo.Session, i *discordgo.InteractionCreate, ...)", signature(fn)))
	}

	if valOf.Type().In(2).Kind() != reflect.Struct {
		panic(fmt.Sprintf("given function(%s) has incorrect type, expecting the 3rd type to be struct not %s", signature(fn), valOf.Type().In(2).Kind().String()))
	}
	e.ty = valOf.Type().In(2)
	e.bindings = generateBindings(e.ty)
	return &e
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

func (e *Executor) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	f := reflect.ValueOf(e.fn)
	d, ok := i.Data.(discordgo.ApplicationCommandInteractionData)
	if ok {
		//todo unguarded type assert
	}
	v := generateExecutorValue(s, d.Options, i.GuildID, e)
	f.Call([]reflect.Value{reflect.ValueOf(s), reflect.ValueOf(i), v})
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
	e.m.Lock()
	defer e.m.Unlock()
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
