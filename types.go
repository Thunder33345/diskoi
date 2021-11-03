package diskoi

import (
	"github.com/bwmarrin/discordgo"
)

type Command interface {
	Name() string
	Description() string
	executor(d discordgo.ApplicationCommandInteractionData) ( //todo remove chain
		executor *Executor,
		chain Chain,
		options []*discordgo.ApplicationCommandInteractionDataOption,
		path []string,
		err error,
	)
	applicationCommand() *discordgo.ApplicationCommand
}

//Mentionable is an instance of something that could be a Role or a User
type Mentionable struct {
	Value interface{}
}

func (m *Mentionable) AsUser() (*discordgo.User, bool) {
	u, ok := m.Value.(*discordgo.User)
	return u, ok
}

func (m *Mentionable) AsRole() (*discordgo.Role, bool) {
	r, ok := m.Value.(*discordgo.Role)
	return r, ok
}

type errorHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, cmd Command, err error)

type rawInteractionHandler func(*discordgo.Session, *discordgo.InteractionCreate)
