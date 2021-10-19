package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"strings"
)

func errPath(path []string) string {
	return "/" + strings.Join(path, " ")
}

type MissingOptionsError struct {
	path []string
}

func (e MissingOptionsError) Error() string {
	return "Missing options(possible api de-sync?): expecting options given for command group, none given for" + errPath(e.path)
}

type NonCommandOptionTypeError struct {
	ty   discordgo.ApplicationCommandOptionType
	path []string
}

func (e NonCommandOptionTypeError) Error() string {
	return "Non command option type(possible api de-sync?): expecting \"SubCommand\" or \"SubCommandGroup\" command option type" +
		" but received \"" + e.ty.String() + "\" for" + errPath(e.path)
}

type MissingSubcommandGroupError struct {
	name string
	path []string
}

func (e MissingSubcommandGroupError) Error() string {
	return "Missing Subcommand group: group \"" + e.name + "\" not found on" + errPath(e.path)
}

type MissingSubcommandError struct {
	name string
	path []string
}

func (e MissingSubcommandError) Error() string {
	return "Missing Subcommand: subcommand \"" + e.name + "\" not found on" + errPath(e.path)
}
