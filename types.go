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

type errorHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, cmd Command, err error)

type rawInteractionHandler func(*discordgo.Session, *discordgo.InteractionCreate)
