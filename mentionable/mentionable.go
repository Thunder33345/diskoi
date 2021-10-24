package mentionable

import "github.com/bwmarrin/discordgo"

type Mentionable struct {
	Role *discordgo.Role
	User *discordgo.User
}
