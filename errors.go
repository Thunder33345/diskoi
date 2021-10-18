package diskoi

import "github.com/bwmarrin/discordgo"

type InteractionDataTypeError struct {
	ty discordgo.InteractionType
}

func (e InteractionDataTypeError) Error() string {
	return "Incorrect interaction data type given, expecting \"ApplicationCommand\"" +
		" but \"" + e.ty.String() + "\" given"
}

type UnreachableError struct {
}

func (e UnreachableError) Error() string {
	return "Unreachable condition reached: possible api command groups de-sync?"
}
