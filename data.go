package diskoi

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
)

//Data is an analysis produced by analyzeCmdFn
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
	opt []*discordgo.ApplicationCommandInteractionDataOption, data *metaArgument) error {
	values, err := reconstructFunctionArgs(d.fnArg, d.cmdArg, d.cmdSpecialArg, data, s, i, opt)
	if err != nil {
		return fmt.Errorf("error reconstructing command: %w", err)
	}
	fn := reflect.ValueOf(d.fn)
	fn.Call(values)
	return nil
}

func (d *Data) Autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate,
	opts []*discordgo.ApplicationCommandInteractionDataOption, data *metaArgument) ([]*discordgo.ApplicationCommandOptionChoice, error) {
	arg, values, err := reconstructAutocompleteArgs(d.cmdArg, d.cmdSpecialArg, data, s, i, opts)
	if err != nil {
		return nil, fmt.Errorf("error autocompleting command: %w", err)
	}
	rets := reflect.ValueOf(arg.autocompleteFn).Call(values)
	optChoice := rets[0].Interface().([]*discordgo.ApplicationCommandOptionChoice)
	return optChoice, nil
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
		return fmt.Errorf("error finding field named %s", fieldName)
	}

	fnArgs, err := analyzeAutocompleteFunction(fn, d.cmdStruct)
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
