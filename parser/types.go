package parser

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
)

type fnArgument struct {
	typ        fnArgumentType
	reflectTyp reflect.Type
}

type fnArgumentType uint8

const (
	fnArgumentTypeInvalid fnArgumentType = iota
	fnArgumentTypeSession
	fnArgumentTypeInteraction
	fnArgumentTypeMarshal
	fnArgumentTypeMarshalPtr
)

func (a fnArgumentType) String() string {
	switch a {
	case fnArgumentTypeInvalid:
		return "Invalid"
	case fnArgumentTypeSession:
		return "Session"
	case fnArgumentTypeInteraction:
		return "InteractionCreate"
	case fnArgumentTypeMarshal:
		return "DiskoiMarshal"
	default:
		return fmt.Sprintf("fnArgumentType(%d)", a)
	}
}

type PayloadArgument struct { //maybe rename to command argument
	fieldIndex []int
	fieldName  string

	cType        discordgo.ApplicationCommandOptionType
	Name         string
	Description  string
	Required     bool
	Choices      []*discordgo.ApplicationCommandOptionChoice
	ChannelTypes []discordgo.ChannelType

	//autocomplete is unimplemented
	autocomplete interface{}
}

type specialArgument struct {
	fieldIndex []int
	fieldName  string
	dataType   specialArgType
}

type specialArgType uint8

const (
	cmdDataTypeDiskoiPath specialArgType = iota
)

type DiskoiData struct {
	path []string
}

type Unmarshal interface {
	UnmarshalDiskoi(s *discordgo.Session, i *discordgo.InteractionCreate,
		o []*discordgo.ApplicationCommandInteractionDataOption) error
}

type ChannelType interface {
	DiskoiChannelTypes() []discordgo.ChannelType
}

type CommandOptions interface {
	DiskoiCommandOptions() []*discordgo.ApplicationCommandOptionChoice
}
