package parser

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
)

//Data is an analysis produced by AnalyzeCmdFn
//It contains everything needed to reconstruct and call it back
//todo rename this Data feels too vague, maybe CommandData
type Data struct {
	//fn is the callback function
	fn interface{}
	//fnArg is a slice of arguments taken by the function
	fnArg []*fnArgument
	//cmdStruct is the struct that will be parsed as the arguments for discord
	//cmdStruct must be the last argument of the fn, if exist
	cmdStruct reflect.Type
	//cmdArg is a slice command arguments from the struct, used for generating ApplicationCommandOptions
	cmdArg []*CommandArgument
	//cmdSpecialArg holds slice of special meta arguments
	cmdSpecialArg []*specialArgument
}

func (d *Data) Execute(s *discordgo.Session, i *discordgo.InteractionCreate,
	opt []*discordgo.ApplicationCommandInteractionDataOption, data *DiskoiData) error {
	values, err := reconstructFunctionArgs(d.fnArg, d.cmdArg, d.cmdSpecialArg, data, s, i, opt)
	if err != nil {
		return fmt.Errorf("error reconstructing command: %w", err)
	}
	fn := reflect.ValueOf(d.fn)
	fn.Call(values)
	return nil
}

func (d *Data) Autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate,
	opts []*discordgo.ApplicationCommandInteractionDataOption, data *DiskoiData) ([]*discordgo.ApplicationCommandOptionChoice, error) {
	//todo move this method into a func
	find := func(name string) *CommandArgument {
		for _, arg := range d.cmdArg {
			if arg.Name == name {
				return arg
			}
		}
		return nil
	}
	for _, opt := range opts {
		if !opt.Focused {
			continue
		}
		arg := find(opt.Name)
		if arg == nil {
			return nil, fmt.Errorf("missing option %s with type %v", opt.Name, opt.Type)
		}
		if arg.cType != opt.Type {
			return nil, fmt.Errorf(`option missmatch in %s: we expect it to be "%v", but discord says it is "%v"`,
				arg.fieldName, arg.cType, opt.Type)
		}
		values, err := reconstructFunctionArgs(arg.autocompleteArgs, d.cmdArg, d.cmdSpecialArg, data, s, i, opts)
		if err != nil {
			return nil, fmt.Errorf("error reconstructing autocomplete: %w", err)
		}
		rets := reflect.ValueOf(arg.autocompleteFn).Call(values)
		optChoice := rets[0].Interface().([]*discordgo.ApplicationCommandOptionChoice)
		return optChoice, nil
	}
	return nil, fmt.Errorf("no options in focus")
}

func (d *Data) AddAutoComplete(fieldName string, fn interface{}) error {
	find := func() *CommandArgument {
		for _, arg := range d.cmdArg {
			if arg.fieldName == fieldName {
				return arg
			}
		}
		return nil
	}
	arg := find()
	if arg == nil {
		//todo error not found
	}

	fnArgs, err := analyzeAutocompleteFunction(fn, d.cmdStruct) //todo use a diff analyzer since it needs returns of autocomplete results
	if err != nil {
		return fmt.Errorf("error analyzing autocomplete function: %w", err)
	}

	arg.autocompleteFn = fn
	arg.autocompleteArgs = fnArgs
	return nil
}

func (d *Data) ApplicationCommandOptions() []*discordgo.ApplicationCommandOption {
	o := make([]*discordgo.ApplicationCommandOption, 0, len(d.cmdArg))
	for _, b := range d.cmdArg {
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

func (d *Data) ArgumentByName(name string) *CommandArgument {
	for _, cmd := range d.cmdArg {
		if cmd.fieldName == name {
			return cmd
		}
	}
	return nil
}

func (d *Data) ArgumentByIndex(index []int) *CommandArgument {
	for _, cmd := range d.cmdArg {
		if indexEqual(cmd.fieldIndex, index) {
			return cmd
		}
	}
	return nil
}

func indexEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
