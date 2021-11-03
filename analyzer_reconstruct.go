package diskoi

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
)

func reconstructFunctionArgs(fnArg []*fnArgument, cmdArg []*commandArgument, data *MetaArgument, ctx context.Context,
	s *discordgo.Session, i *discordgo.InteractionCreate,
	o []*discordgo.ApplicationCommandInteractionDataOption) ([]reflect.Value, error) {
	values := make([]reflect.Value, 0, len(fnArg))
	for _, arg := range fnArg {
		switch arg.typ {
		case fnArgumentTypeSession:
			values = append(values, reflect.ValueOf(s))
		case fnArgumentTypeInteraction:
			values = append(values, reflect.ValueOf(i))
		case fnArgumentTypeData:
			v, err := reconstructCommandArgument(arg.reflectTyp, cmdArg, s, i, o)
			if err != nil {
				return nil, fmt.Errorf(`reconstructing command data "%s": %w`, arg.reflectTyp.String(), err)
			}
			values = append(values, v)
		case fnArgumentTypeMeta:
			values = append(values, reflect.ValueOf(data))
		case fnArgumentTypeContext:
			values = append(values, reflect.ValueOf(ctx))
		case fnArgumentTypeMarshal, fnArgumentTypeMarshalPtr:
			mt := reflect.New(arg.reflectTyp)
			m := mt.Interface().(Unmarshal)
			err := m.UnmarshalDiskoi(s, i, o)
			if err != nil {
				return nil, fmt.Errorf("unmarshalling %s: %w", arg.reflectTyp.String(), err)
			}
			if arg.typ == fnArgumentTypeMarshalPtr {
				values = append(values, reflect.ValueOf(m))
			} else {
				values = append(values, reflect.ValueOf(m).Elem())
			}
		default:
			return nil, fmt.Errorf("unrecognized argument type #%d (%s)", uint(arg.typ), arg.typ.String())
		}
	}
	return values, nil
}

func reconstructAutocompleteArgs(cmdArg []*commandArgument, data *MetaArgument,
	s *discordgo.Session, i *discordgo.InteractionCreate,
	opts []*discordgo.ApplicationCommandInteractionDataOption) (*commandArgument, []reflect.Value, error) {
	for _, opt := range opts {
		if !opt.Focused {
			continue
		}
		arg := findCmdArg(cmdArg, opt.Name)
		if arg == nil {
			return nil, nil, fmt.Errorf(`cant find option named "%s" type of "%v" locally`, opt.Name, opt.Type)
		}
		if arg.cType != opt.Type {
			return nil, nil, newDiscordExpectationError(fmt.Sprintf(`option mismatch in %s: we expect it to be "%v", but discord says it is "%v"`,
				arg.fieldName, arg.cType, opt.Type))
		}

		values, err := reconstructFunctionArgs(arg.autocompleteArgs, cmdArg, data, context.Background(), s, i, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("reconstructing autocomplete: %w", err)
		}
		return arg, values, nil
	}
	return nil, nil, newDiscordExpectationError(fmt.Sprintf("no options in focus"))
}

func reconstructCommandArgument(cmdStruct reflect.Type, cmdArg []*commandArgument,
	s *discordgo.Session, i *discordgo.InteractionCreate,
	opts []*discordgo.ApplicationCommandInteractionDataOption) (reflect.Value, error) {
	val := reflect.New(cmdStruct)
	if cmdStruct.Kind() != reflect.Ptr {
		val = val.Elem()
	}

	for _, opt := range opts {
		py := findCmdArg(cmdArg, opt.Name)
		if py == nil {
			return reflect.Value{}, fmt.Errorf(`cant find option named "%s" type of "%v" locally`, opt.Name, opt.Type)
		}
		if py.cType != opt.Type {
			return reflect.Value{}, newDiscordExpectationError(fmt.Sprintf(`option type mismatch in "%s": we expect it to be "%v", but discord says it is "%v"`,
				py.fieldName, py.cType, opt.Type))
		}
		fVal := val.FieldByIndex(py.fieldIndex)
		var v interface{}
		switch opt.Type {
		case discordgo.ApplicationCommandOptionString:
			x := opt.StringValue()
			v = &x
		case discordgo.ApplicationCommandOptionInteger:
			switch fVal.Kind() {
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				x := opt.UintValue()
				v = &x
			default:
				x := opt.IntValue()
				v = &x
			}
		case discordgo.ApplicationCommandOptionBoolean:
			x := opt.BoolValue()
			v = &x
		case 10: //type doubles
			x := opt.FloatValue()
			v = &x
		case discordgo.ApplicationCommandOptionChannel:
			v = opt.ChannelValue(s)
		case discordgo.ApplicationCommandOptionUser:
			v = opt.UserValue(s)
		case discordgo.ApplicationCommandOptionRole:
			v = opt.RoleValue(s, i.GuildID)
		case discordgo.ApplicationCommandOptionMentionable:
			men := &Mentionable{}
			if u, err := s.User(opt.Value.(string)); err == nil {
				men.Value = u
			} else if r, err := s.State.Role(i.GuildID, opt.Value.(string)); err == nil {
				men.Value = r
			}
			v = men
		default:
			return reflect.Value{}, newDiscordExpectationError(fmt.Sprintf(`unrecognized ApplicationCommandOptionType "%v" in "%s"`, opt.Type, py.fieldName))
		}
		recVal := reflect.ValueOf(v)
		if fVal.Kind() != reflect.Ptr {
			recVal = recVal.Elem()
		}
		if fVal.Kind() != recVal.Kind() {
			if recVal.CanConvert(fVal.Type()) {
				recVal = recVal.Convert(fVal.Type())
			} else {
				return reflect.Value{}, fmt.Errorf(`cant convert %s(%v) into %s(%v)`,
					recVal.Type().String(), recVal.Type().Kind(), fVal.Type().String(), fVal.Type().Kind())
			}
		}
		fVal.Set(recVal)
	}
	return val, nil
}

func findCmdArg(cmdArgs []*commandArgument, name string) *commandArgument {
	for _, arg := range cmdArgs {
		if name == arg.Name {
			return arg
		}
	}
	return nil
}
