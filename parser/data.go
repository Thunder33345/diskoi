package parser

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
)

//Data is an analysis produced by Analyze
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
	args, err := reconstructFunctionArgs(d.fnArg, s, i, opt)
	if err != nil {
		return errors.New(fmt.Sprintf("error reconstructing arguments: %s", err.Error()))
	}

	if d.cmdStruct != nil {
		py, err := reconstructCommandArgument(d, s, i, opt, data)
		if err != nil {
			return errors.New(fmt.Sprintf("error reconstructing command argument %s: %s", d.cmdStruct.String(), err.Error()))
		}
		args = append(args, py)
	}
	fn := reflect.ValueOf(d.fn)
	fn.Call(args)
	return nil
}

func (d *Data) Autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate,
	options []*discordgo.ApplicationCommandInteractionDataOption, data DiskoiData) error {
	//todo scaffold for autocompletion
	panic("TODO: Autocomplete")
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
