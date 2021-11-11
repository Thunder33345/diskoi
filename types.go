package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"sync"
)

type Command interface {
	Name() string
	Description() string
	execute(s *discordgo.Session, i *discordgo.InteractionCreate, pre Chain) error
	autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) ([]*discordgo.ApplicationCommandOptionChoice, error)
	applicationCommand() *discordgo.ApplicationCommand
	lock()
}

//Mentionable is an instance of something that could be a Role or a User
type Mentionable struct {
	Value interface{}
}

func (m *Mentionable) AsUser() (*discordgo.User, bool) {
	u, ok := m.Value.(*discordgo.User)
	return u, ok
}

func (m *Mentionable) AsRole() (*discordgo.Role, bool) {
	r, ok := m.Value.(*discordgo.Role)
	return r, ok
}

type errorHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, cmd Command, err error)

type rawInteractionHandler func(*discordgo.Session, *discordgo.InteractionCreate)

func withRWMutex(m *sync.RWMutex, fn func()) {
	m.RLock()
	defer m.RUnlock()
	fn()
}

type registerMapping struct {
	command Command
	guild   string
}
