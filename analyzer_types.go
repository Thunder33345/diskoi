package diskoi

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"strings"
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
	fnArgumentTypeData
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

type CommandArgument struct {
	fieldIndex []int
	fieldName  string

	cType        discordgo.ApplicationCommandOptionType
	Name         string
	Description  string
	Required     bool
	Choices      []*discordgo.ApplicationCommandOptionChoice
	ChannelTypes []discordgo.ChannelType

	autocompleteFn   interface{}
	autocompleteArgs []*fnArgument
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

type metaArgument struct {
	Path []string
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

//https://stackoverflow.com/questions/54129042/how-to-get-a-functions-signature-as-string-in-go
func signature(f interface{}) string {
	t := reflect.TypeOf(f)
	if t.Kind() != reflect.Func {
		return fmt.Sprintf("<not a function(is %s)>", t.Kind().String())
	}

	buf := strings.Builder{}
	buf.WriteString("func (")
	for i := 0; i < t.NumIn(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(t.In(i).String())
	}
	buf.WriteString(")")
	if numOut := t.NumOut(); numOut > 0 {
		if numOut > 1 {
			buf.WriteString(" (")
		} else {
			buf.WriteString(" ")
		}
		for i := 0; i < t.NumOut(); i++ {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(t.Out(i).String())
		}
		if numOut > 1 {
			buf.WriteString(")")
		}
	}

	return buf.String()
}
