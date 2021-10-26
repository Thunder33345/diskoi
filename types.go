package diskoi

import (
	"github.com/bwmarrin/discordgo"
)

type Command interface {
	Name() string
	Description() string
	executor(d discordgo.ApplicationCommandInteractionData) (
		executor *Executor,
		options []*discordgo.ApplicationCommandInteractionDataOption,
		path []string,
		err error,
	)
	applicationCommand() *discordgo.ApplicationCommand
}

//Mentionable is an instance of something that could be a Role or a User
//todo improve this type into something like an interface receiver and access functions rather then 2 values
type Mentionable struct {
	Role *discordgo.Role
	User *discordgo.User
}

type errorHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, cmd Command, err error)

type rawInteractionHandler func(*discordgo.Session, *discordgo.InteractionCreate)
