package diskoi

import "github.com/bwmarrin/discordgo"

type MissingOptionsError struct {
}

func (e MissingOptionsError) Error() string {
	return "Missing options: expecting options given for command group, none given(possible api de-sync?)"
}

type NonCommandOptionTypeError struct {
	ty discordgo.ApplicationCommandOptionType
}

func (e NonCommandOptionTypeError) Error() string {
	return "Unexpected interaction data type given, expecting \"SubCommand\" or \"SubCommandGroup\"" +
		" but received \"" + e.ty.String() + "\""
}

type MissingSubcommandGroupError struct {
	name string
}

func (e MissingSubcommandGroupError) Error() string {
	return "Missing Subcommand group: group \"" + e.name + "\" not found"
}

type MissingSubcommandError struct {
	name string
}

func (e MissingSubcommandError) Error() string {
	return "Missing Subcommand: subcommand \"" + e.name + "\" not found"
}
