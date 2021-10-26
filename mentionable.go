package diskoi

import "github.com/bwmarrin/discordgo"

//Mentionable is an instance of something that could be a Role or a User
//todo refactor everything into ../ someday, currently isolated to prevent import cycles
type Mentionable struct {
	Role *discordgo.Role
	User *discordgo.User
}
