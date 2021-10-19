package diskoi

func (d *Diskoi) RegisterCommands() error {
	d.m.Lock()
	defer d.m.Unlock()
	s := d.s
	for _, cmd := range d.commands {
		cc, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd.applicationCommand())
		if err != nil {
			return err
		}
		d.registeredCommand[cc.ID] = cmd
	}
	for gid, cms := range d.commandsGuild {
		for _, cmd := range cms {
			cc, err := s.ApplicationCommandCreate(s.State.User.ID, gid, cmd.applicationCommand())
			if err != nil {
				return err
			}
			d.registeredCommand[cc.ID] = cmd
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

func (d *Diskoi) RemoveGuildCommand(guild string, exec executable) error {
	d.m.Lock()
	defer d.m.Unlock()
	if guild == "" {
		for i, e2 := range d.commands {
			if exec == e2 {
				d.commands = append(d.commands[:i], d.commands[i+1:]...)
			}
		}
	} else {
		for i, e2 := range d.commandsGuild[guild] {
			if exec == e2 {
				d.commands = append(d.commandsGuild[guild][:i], d.commandsGuild[guild][i+1:]...)
			}
		}
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
