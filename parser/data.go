package parser

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
)

//Data is an analysis produced by Analyze
//It contains everything needed to reconstruct and call it back
type Data struct {
	//fn is the callback function
	fn interface{}
	//fnArg is a slice of analyzed arguments taken by the function
	fnArg []*fnArgument
	//pyTy is type of the payload for this fn
	//payload must be the last argument of the fn, if exist
	//payload is the struct that will be present to discord as command options
	pyTy reflect.Type
	//pyArg is a slice of analyzed payload data, used for generating ApplicationCommandOptions
	pyArg []*PayloadArgument
	//pysArg holds slice of special meta arguments
	pysArg []*specialArgument
}

func (d *Data) Execute(s *discordgo.Session, i *discordgo.InteractionCreate,
	opt []*discordgo.ApplicationCommandInteractionDataOption, data DiskoiData) error {
	args, err := reconstructFunctionArgs(d.fnArg, s, i, opt)
	if err != nil {
		return errors.New(fmt.Sprintf("error reconstructing arguments: %s", err.Error()))
	}

	if d.pyTy != nil {
		py, err := reconstructPayload(d, s, i, opt, data)
		if err != nil {
			return errors.New(fmt.Sprintf("error reconstructing payload %s: %s", d.pyTy.Name(), err.Error()))
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
	o := make([]*discordgo.ApplicationCommandOption, 0, len(d.pyArg))
	for _, b := range d.pyArg {
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

func (d *Data) ArgumentByName(name string) *PayloadArgument {
	for _, cmd := range d.pyArg {
		if cmd.fieldName == name {
			return cmd
		}
	}
	return nil
}

func (d *Data) ArgumentByIndex(index []int) *PayloadArgument {
	for _, cmd := range d.pyArg {
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
