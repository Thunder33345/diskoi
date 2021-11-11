package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"reflect"
)

func (d *Diskoi) RegisterCommands() error { //todo allow selective registering between guild or global
	d.m.Lock()
	defer d.m.Unlock()
	s := d.s
	f := func(c Command, g string) error {
		cc, err := s.ApplicationCommandCreate(s.State.User.ID, g, c.applicationCommand())
		if err != nil {
			return DiscordAPIError{err: err}
		}
		d.registeredCommand[cc.ID] = registerMapping{
			command: c,
			guild:   g,
		}
		return nil
	}
	for _, cmd := range d.commands {
		err := f(cmd, "")
		if err != nil {
			return err
		}
	}
	for gid, cms := range d.commandsGuild {
		for _, cmd := range cms {
			err := f(cmd, gid)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Diskoi) SyncCommands() error {
	d.m.Lock()
	defer d.m.Unlock()
	f := func(guild string, cs []Command) error {
		rc, err := d.s.ApplicationCommands(d.s.State.User.ID, guild)
		if err != nil {
			return DiscordAPIError{err: err}
		}
		cMap := make(map[string]*discordgo.ApplicationCommand, len(rc))
		for _, cmd := range rc {
			if cmd.Type != discordgo.ChatApplicationCommand {
				continue
			}
			cMap[cmd.Name] = cmd
		}

		eMap := make(map[string]struct{}, len(cs))
		for _, c := range cs {
			eMap[c.Name()] = struct{}{}
			rc, ok := cMap[c.Name()]
			eac := c.applicationCommand()
			if ok {
				if len(eac.Options) == len(rc.Options) &&
					eac.Description == rc.Description &&
					(len(eac.Options) == 0 || reflect.DeepEqual(eac.Options, rc.Options)) {
					d.registeredCommand[rc.ID] = registerMapping{
						command: c,
						guild:   guild,
					}
					continue
				}
			}
			cc, err := d.s.ApplicationCommandCreate(d.s.State.User.ID, guild, eac)
			if err != nil {
				return DiscordAPIError{err: err}
			}
			d.registeredCommand[cc.ID] = registerMapping{
				command: c,
				guild:   guild,
			}
		}

		for cName, cmd := range cMap {
			_, ok := eMap[cName]
			if !ok {
				err = d.s.ApplicationCommandDelete(d.s.State.User.ID, guild, cmd.ID)
				if err != nil {
					return DiscordAPIError{err: err}
				}
			}
		}
		return nil
	}
	err := f("", d.commands)
	if err != nil {
		return err
	}
	for guild := range d.commandsGuild {
		err := f(guild, d.commandsGuild[guild])
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Diskoi) UnregisterAllCommands() error {
	d.m.Lock()
	defer d.m.Unlock()
	s := d.s
	for id := range d.registeredCommand {
		err := s.ApplicationCommandDelete(s.State.User.ID, "", id)
		if err != nil {
			return DiscordAPIError{err: err}
		}
		delete(d.registeredCommand, id)
	}
	return nil
}

func (d *Diskoi) UnregisterCommands(guild string) error {
	d.m.Lock()
	defer d.m.Unlock()
	s := d.s
	for id, rCmd := range d.registeredCommand {
		if rCmd.guild != guild {
			continue
		}
		err := s.ApplicationCommandDelete(s.State.User.ID, "", id)
		if err != nil {
			return DiscordAPIError{err: err}
		}
		delete(d.registeredCommand, id)
	}
	return nil
}

func (d *Diskoi) AddCommand(cmd Command) {
	d.AddGuildCommand("", cmd)
}

func (d *Diskoi) AddGuildCommand(guild string, cmd Command) {
	d.m.Lock()
	defer d.m.Unlock()
	cmd.lock()

	dupe, i := d.findGuildCommandByName(guild, cmd.Name())
	if dupe != nil {
		c, id := d.findRegisteredCmdUnsafe(dupe)
		if c != nil {
			delete(d.registeredCommand, id)
		}

		if guild == "" {
			d.commands[i] = cmd
		} else {
			d.commandsGuild[guild][i] = cmd
		}
	}
	if guild == "" {
		d.commands = append(d.commands, cmd)
	} else {
		d.commandsGuild[guild] = append(d.commandsGuild[guild], cmd)
	}
}

func (d *Diskoi) RemoveCommand(cmd Command) error {
	return d.RemoveGuildCommand("", cmd)
}

func (d *Diskoi) RemoveGuildCommand(guild string, cmd Command) error {
	d.m.Lock()
	defer d.m.Unlock()
	f := func(cs []Command, cmd Command) []Command {
		for i := 0; i < len(cs); {
			v := cs[i]
			if v == cmd {
				cs = append(cs[:i], cs[i+1:]...)
				continue
			}
			i++
		}
		return cs
	}

	if guild == "" {
		d.commands = f(d.commands, cmd)
	} else {
		d.commandsGuild[guild] = f(d.commandsGuild[guild], cmd)
	}

	for id, e2 := range d.registeredCommand {
		if cmd == e2.command {
			err := d.s.ApplicationCommandDelete(d.s.State.User.ID, guild, id)
			if err != nil {
				return DiscordAPIError{err: err}
			}
		}
	}
	return nil
}

func (d *Diskoi) FindCommandByName(name string) Command {
	d.m.Lock()
	defer d.m.Unlock()
	c, _ := d.findGuildCommandByName("", name)
	return c
}

func (d *Diskoi) FindGuildCommandByName(guild string, name string) Command {
	d.m.Lock()
	defer d.m.Unlock()
	c, _ := d.findGuildCommandByName(guild, name)
	return c
}

func (d *Diskoi) findGuildCommandByName(guild string, name string) (Command, int) {
	f := func(c []Command) (Command, int) {
		for i, cmd := range c {
			if cmd.Name() == name {
				return cmd, i
			}
		}
		return nil, -1
	}

	if guild == "" {
		return f(d.commands)
	}
	return f(d.commandsGuild[guild])
}

func (d *Diskoi) findRegisteredCmdUnsafe(cmd Command) (Command, string) {
	for id, rc := range d.registeredCommand {
		if cmd == rc.command {
			return cmd, id
		}
	}
	return nil, ""
}
