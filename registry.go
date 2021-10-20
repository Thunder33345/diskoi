package diskoi

import "github.com/bwmarrin/discordgo"

func (d *Diskoi) RegisterCommands() error {
	d.m.Lock()
	defer d.m.Unlock()
	s := d.s
	f := func(e executable, g string) error {
		cc, err := s.ApplicationCommandCreate(s.State.User.ID, g, e.applicationCommand())
		if err != nil {
			return err
		}
		d.registeredCommand[cc.ID] = e
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

func (d *Diskoi) AssumeRegistered() error {
	d.m.Lock()
	defer d.m.Unlock()
	f := func(guild string) error { //todo compare registered vs inventory
		rc, err := d.s.ApplicationCommands(d.s.State.User.ID, guild)
		if err != nil {
			return err
		}
		for _, cmd := range rc {
			if cmd.Type != discordgo.ChatApplicationCommand {
				continue
			}

			exec := d.findGuildCommandByName(guild, cmd.Name)
			if exec == nil {
				_ = d.s.ApplicationCommandDelete(d.s.State.User.ID, guild, cmd.ID)
				continue
			}
			if len(cmd.Options) == len(exec.applicationCommand().Options) { //todo better comparing
				d.registeredCommand[cmd.ID] = exec
			} else {
				cc, err := d.s.ApplicationCommandCreate(d.s.State.User.ID, guild, exec.applicationCommand())
				if err != nil {
					return err
				}
				d.registeredCommand[cc.ID] = exec
			}
		}
		return nil
	}
	err := f("")
	if err != nil {
		return err
	}
	for guild := range d.commandsGuild {
		err := f(guild)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Diskoi) UnregisterCommands() error {
	d.m.Lock()
	defer d.m.Unlock()
	s := d.s
	for id := range d.registeredCommand {
		err := s.ApplicationCommandDelete(s.State.User.ID, "735145518220443680", id)
		if err != nil {
			return err
		}
		delete(d.registeredCommand, id)
	}
	return nil
}

func (d *Diskoi) AddCommand(exec executable) {
	d.AddGuildCommand("", exec)
}

func (d *Diskoi) AddGuildCommand(guild string, exec executable) {
	d.m.Lock()
	defer d.m.Unlock()
	//todo duplicate = upsert
	if guild == "" {
		d.commands = append(d.commands, exec)
	} else {
		d.commandsGuild[guild] = append(d.commandsGuild[guild], exec)
	}
}

func (d *Diskoi) RemoveCommand(exec executable) error {
	return d.RemoveGuildCommand("", exec)
}

func (d *Diskoi) RemoveGuildCommand(guild string, exec executable) error {
	d.m.Lock()
	defer d.m.Unlock()
	f := func(es []executable, exec executable) []executable {
		for i := 0; i < len(es); {
			v := es[i]
			if v == exec {
				es = append(es[:i], es[i+1:]...)
				continue
			}
			i++
		}
		return es
	}

	if guild == "" {
		d.commands = f(d.commands, exec)
	} else {
		d.commandsGuild[guild] = f(d.commandsGuild[guild], exec)
	}

	for id, e2 := range d.registeredCommand {
		if exec == e2 {
			err := d.s.ApplicationCommandDelete(d.s.State.User.ID, guild, id)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Diskoi) findGuildCommandByName(guild string, name string) executable { //idea consider exporting
	f := func(e []executable) executable {
		for _, cmd := range e {
			if cmd.Name() == name {
				return cmd
			}
		}
		return nil
	}

	if guild == "" {
		return f(d.commands)
	}
	return f(d.commandsGuild[guild])
}
