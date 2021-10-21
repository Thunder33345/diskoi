package diskoi

import "github.com/bwmarrin/discordgo"

type commandBinding struct {
	fieldIndex int
	fieldName  string

	cType        discordgo.ApplicationCommandOptionType
	name         string
	description  string
	required     bool
	choices      []*discordgo.ApplicationCommandOptionChoice
	channelTypes []discordgo.ChannelType

	//autocomplete is a callback to autocomplete for this option
	//will fire this if this is focused
	//unimplemented does nothing
	autocomplete interface{}
}
