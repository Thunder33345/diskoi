package diskoi

import "github.com/bwmarrin/discordgo"

type commandBinding struct {
	FieldIndex int
	FieldName  string

	Type         discordgo.ApplicationCommandOptionType
	Name         string
	Description  string
	Required     bool
	Choices      []*discordgo.ApplicationCommandOptionChoice
	ChannelTypes []discordgo.ChannelType

	//autocomplete is a callback to autocomplete for this option
	//will fire this if this is focused
	//unimplemented does nothing
	autocomplete interface{}
}