package diskoi

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"strings"
)

type executable interface {
	Name() string
	executor(d discordgo.ApplicationCommandInteractionData) (
		executor *Executor,
		options []*discordgo.ApplicationCommandInteractionDataOption,
		err error,
	)
	applicationCommand() *discordgo.ApplicationCommand
}

type errorHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, exec executable, err error)

type rawInteractionHandler func(*discordgo.Session, *discordgo.InteractionCreate)

type Mentionable struct {
	Role *discordgo.Role
	User *discordgo.User
}

type applicationCommandLister interface {
	applicationCommand() *discordgo.ApplicationCommand
}

type applicationCommandOptionsLister interface {
	applicationCommandOptions() []*discordgo.ApplicationCommandOption
}

type applicationCommandOptionLister interface {
	applicationCommandOption() *discordgo.ApplicationCommandOption
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
