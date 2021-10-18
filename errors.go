package diskoi

import "github.com/bwmarrin/discordgo"

type InteractionDataTypeError struct {
	ty discordgo.InteractionType
}

func (e InteractionDataTypeError) Error() string {
	return "Incorrect interaction data type given, expecting \"ApplicationCommand\"" +
		" but \"" + e.ty.String() + "\" given"
}

type MissingSubcommandError struct {
}

func (e MissingSubcommandError) Error() string {
	return "Subcommand not found: possible api command groups de-sync?"
}
