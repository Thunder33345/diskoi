package diskoi

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"strconv"
	"strings"
)

func generateExecutorValue(
	s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption, executor *Executor,
) (reflect.Value, error) {
	valO := reflect.New(executor.ty)
	val := valO.Elem()
	findBindings := func(name string) *commandBinding {
		for _, b := range executor.bindings {
			if name == b.name {
				return b
			}
		}
		return nil
	}
	for _, opt := range options {
		b := findBindings(opt.Name)
		if b == nil {
			return reflect.Value{}, MissingBindingsError{name: opt.Name}
		}
		vf := val.Field(b.fieldIndex)
		var v interface{}
		switch b.cType {
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
			v = opt.RoleValue(s, i.GuildID)
		case discordgo.ApplicationCommandOptionMentionable:
			u, err := s.User(opt.Value.(string))
			if err == nil {
				vf.FieldByName("User").Set(reflect.ValueOf(u))
				continue
			}
			r, err := s.State.Role(i.GuildID, opt.Value.(string))
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
	return val, nil
}

func generateBindings(t reflect.Type) ([]*commandBinding, error) {
	if t.Kind() != reflect.Struct {
		return nil, errors.New(fmt.Sprintf("given type %s(%s) is not type of struct", t.Name(), t.Kind().String()))
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
		bind.fieldIndex = i
		bind.fieldName = f.Name
		bind.name = strings.ToLower(f.Name)

		if ok {
			tags := splitTag(tag)
			for key, val := range tags {
				switch key {
				case "name":
					bind.name = val
				case "description":
					bind.description = val
				case "required":
					bind.required = true
				case "channelTypes":
					cts := strings.Split(val, "+")
					for _, ct := range cts {
						ci, e := strconv.Atoi(ct)
						if e != nil {
							return nil, errors.New(fmt.Sprintf("non int convertable given for channelTypes on %s for %s", f.Name, t.Name()))
						}
						bind.channelTypes = append(bind.channelTypes, discordgo.ChannelType(ci))
					}
				}
			}
		}

		if bind.description == "" {
			return nil, errors.New(fmt.Sprintf("Description of %s.%s cant be empty", t.Name(), bind.fieldName))
		}

		kind := f.Type.Kind()
		if kind == reflect.Ptr {
			kind = f.Type.Elem().Kind()
		}
		switch kind {
		case reflect.String:
			bind.cType = discordgo.ApplicationCommandOptionString
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			bind.cType = discordgo.ApplicationCommandOptionInteger
		case reflect.Bool:
			bind.cType = discordgo.ApplicationCommandOptionBoolean
		case reflect.Float32:
		case reflect.Float64:
			bind.cType = 10 //type doubles fixme get constant from discord go
		case reflect.Struct:
			switch {
			case f.Type == reflect.TypeOf(discordgo.Channel{}):
				bind.cType = discordgo.ApplicationCommandOptionChannel
			case f.Type == reflect.TypeOf(discordgo.User{}):
				bind.cType = discordgo.ApplicationCommandOptionUser
			case f.Type == reflect.TypeOf(discordgo.Role{}):
				bind.cType = discordgo.ApplicationCommandOptionRole
			case f.Type == reflect.TypeOf(Mentionable{}):
				bind.cType = discordgo.ApplicationCommandOptionMentionable
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
		if b.required {
			req = append(req, b)
		} else {
			opt = append(opt, b)
		}
	}
	binds = append(req, opt...)
	return binds, nil
}

func splitTag(tag string) map[string]string {
	split := strings.Split(tag, ",")
	res := make(map[string]string, len(split))
	for _, sub := range split {
		kv := strings.SplitN(sub, ":", 2)
		switch len(kv) {
		default:
			continue
		case 1:
			res[kv[0]] = ""
		case 2:
			res[kv[0]] = kv[1]
		}
	}
	return res
}
