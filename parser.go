package diskoi

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"strconv"
	"strings"
)

func generateExecutorValue(s *discordgo.Session, options []*discordgo.ApplicationCommandInteractionDataOption, guildID string, executor *Executor) reflect.Value {
	valO := reflect.New(executor.ty)
	val := valO.Elem()
	findBindings := func(name string) *commandBinding {
		for _, b := range executor.bindings {
			if name == b.Name {
				return b
			}
		}
		return nil
	}
	for _, opt := range options {
		b := findBindings(opt.Name)
		if b == nil {
			panic("Failed to find " + opt.Name)
		}
		vf := val.Field(b.FieldIndex)
		var v interface{}
		switch b.Type {
		case discordgo.ApplicationCommandOptionString:
			x := opt.StringValue()
			switch {
			case vf.Kind() == reflect.Ptr:
				v = &x
			default:
				v = x
			}
		case discordgo.ApplicationCommandOptionInteger:
			x := opt.IntValue()
			switch {
			case vf.Kind() == reflect.Ptr:
				v = &x
			default:
				v = x
			}
		case discordgo.ApplicationCommandOptionBoolean:
			x := opt.BoolValue()
			switch {
			case vf.Kind() == reflect.Ptr:
				v = &x
			default:
				v = x
			}
		case 10: //type doubles
			x := opt.FloatValue()
			switch {
			case vf.Kind() == reflect.Ptr:
				v = &x
			default:
				v = x
			}
		case discordgo.ApplicationCommandOptionChannel:
			v = opt.ChannelValue(s)
		case discordgo.ApplicationCommandOptionUser:
			v = opt.UserValue(s)
		case discordgo.ApplicationCommandOptionRole:
			v = opt.RoleValue(s, guildID)
		case discordgo.ApplicationCommandOptionMentionable:
			u, err := s.User(opt.Value.(string))
			if err == nil {
				vf.FieldByName("User").Set(reflect.ValueOf(u))
				continue
			}
			r, err := s.State.Role(guildID, opt.Value.(string))
			if err == nil {
				vf.FieldByName("Role").Set(reflect.ValueOf(r))
				continue
			}
			continue
		default:
			continue //skip since we can't process this
		}
		rv := reflect.ValueOf(v)
		switch {
		//case vf.Kind() == reflect.Ptr && rv.Kind() != reflect.Ptr:
		//	rv = rv.Addr()
		case vf.Kind() != reflect.Ptr && rv.Kind() == reflect.Ptr:
			rv = rv.Elem()
		}
		if vf.Kind() != rv.Kind() {
			rv = rv.Convert(vf.Type())
		}
		vf.Set(rv)
	}
	return val
}

func generateBindings(t reflect.Type) []*commandBinding {//todo do something about blank description
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("given interface %s(%s) is not type of struct", t.Name(), t.Kind().String()))
	}

	binds := make([]*commandBinding, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		tag, ok := f.Tag.Lookup(magicTag)
		if tag == "-" {
			continue //temp assuming no tag = ignore
		}

		bind := &commandBinding{}
		bind.FieldIndex = i
		bind.FieldName = f.Name
		bind.Name = strings.ToLower(f.Name)

		if ok {
			tags := splitTag(tag)
			for key, val := range tags {
				switch key {
				case "name":
					bind.Name = val
				case "description":
					bind.Description = val
				case "required":
					bind.Required = true
				case "channelTypes":
					cts := strings.Split(val, "+")
					for _, ct := range cts {
						ci, e := strconv.Atoi(ct)
						if e != nil {
							panic(fmt.Sprintf("non int convertable given for channelTypes on %s for %s", f.Name, t.Name()))
						}
						bind.ChannelTypes = append(bind.ChannelTypes, discordgo.ChannelType(ci))
					}
				}
			}
		}

		kind := f.Type.Kind()
		if kind == reflect.Ptr {
			kind = f.Type.Elem().Kind()
		}
		switch kind {
		case reflect.String:
			bind.Type = discordgo.ApplicationCommandOptionString
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			bind.Type = discordgo.ApplicationCommandOptionInteger
		case reflect.Bool:
			bind.Type = discordgo.ApplicationCommandOptionBoolean
		case reflect.Float32:
		case reflect.Float64:
			bind.Type = 10 //type doubles fixme get constant from discord go
		case reflect.Struct:
			switch {
			case f.Type == reflect.TypeOf(discordgo.Channel{}):
				bind.Type = discordgo.ApplicationCommandOptionChannel
			case f.Type == reflect.TypeOf(discordgo.User{}):
				bind.Type = discordgo.ApplicationCommandOptionUser
			case f.Type == reflect.TypeOf(discordgo.Role{}):
				bind.Type = discordgo.ApplicationCommandOptionRole
			case f.Type == reflect.TypeOf(Mentionable{}):
				bind.Type = discordgo.ApplicationCommandOptionMentionable
			default: //idea: support custom types with diskoi marshaller
				continue
			}
		default:
			continue //skip since we can't process this
		}
		binds = append(binds, bind)
	}
	//sort and prioritize required cuz that's what they api wants us to do
	req := make([]*commandBinding, 0, t.NumField())
	opt := make([]*commandBinding, 0, t.NumField())
	for _, b := range binds {
		if b.Required {
			req = append(req, b)
		} else {
			opt = append(opt, b)
		}
	}
	binds = append(req, opt...)
	return binds
}
