package diskoi

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
)

func reconstructFunctionArgs(fnArg []*fnArgument, cmdArg []*CommandArgument, cmdSpecialArg []*specialArgument, data *specialData,
	s *discordgo.Session, i *discordgo.InteractionCreate,
	o []*discordgo.ApplicationCommandInteractionDataOption) ([]reflect.Value, error) {
	values := make([]reflect.Value, 0, len(fnArg))
	for _, arg := range fnArg {
		switch arg.typ {
		case fnArgumentTypeSession:
			values = append(values, reflect.ValueOf(s))
		case fnArgumentTypeInteraction:
			values = append(values, reflect.ValueOf(i))
		case fnArgumentTypeMarshal, fnArgumentTypeMarshalPtr:
			mt := reflect.New(arg.reflectTyp)
			m := mt.Interface().(Unmarshal)
			err := m.UnmarshalDiskoi(s, i, o)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling %s: %w", arg.reflectTyp.String(), err)
			}
			if arg.typ == fnArgumentTypeMarshalPtr {
				values = append(values, reflect.ValueOf(m))
			} else {
				values = append(values, reflect.ValueOf(m).Elem())
			}
		case fnArgumentTypeData:
			v, err := reconstructCommandArgument(arg.reflectTyp, cmdArg, cmdSpecialArg, s, i, o, data)
			if err != nil {
				return nil, fmt.Errorf(`error reconstructing command data "%s": %w`, arg.reflectTyp.String(), err)
			}
			values = append(values, v)
		default:
			return nil, fmt.Errorf("unrecognized argument type #%d (%s)", uint(arg.typ), arg.typ.String())
		}
	}
	return values, nil
}
func reconstructAutocompleteArgs(cmdArg []*CommandArgument, cmdSpecialArg []*specialArgument, data *specialData,
	s *discordgo.Session, i *discordgo.InteractionCreate,
	opts []*discordgo.ApplicationCommandInteractionDataOption) (*CommandArgument, []reflect.Value, error) {
	find := func(name string) *CommandArgument {
		for _, arg := range cmdArg {
			if arg.Name == name {
				return arg
			}
		}
		return nil
	}
	for _, opt := range opts {
		if !opt.Focused {
			continue
		}
		arg := find(opt.Name)
		if arg == nil {
			return nil, nil, fmt.Errorf("missing option %s with type %v", opt.Name, opt.Type)
		}
		if arg.cType != opt.Type {
			return nil, nil, fmt.Errorf(`option missmatch in %s: we expect it to be "%v", but discord says it is "%v"`,
				arg.fieldName, arg.cType, opt.Type)
		}
		values, err := reconstructFunctionArgs(arg.autocompleteArgs, cmdArg, cmdSpecialArg, data, s, i, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("error reconstructing autocomplete: %w", err)
		}
		return arg, values, nil
	}
	return nil, nil, fmt.Errorf("no options in focus")
}
func reconstructCommandArgument(cmdStruct reflect.Type, cmdArg []*CommandArgument, cmdSpecialArg []*specialArgument,
	s *discordgo.Session, i *discordgo.InteractionCreate,
	opts []*discordgo.ApplicationCommandInteractionDataOption, data *specialData) (reflect.Value, error) {
	val := reflect.New(cmdStruct).Elem()
	for _, opt := range opts {
		py := findPyArg(cmdArg, opt.Name)
		if py == nil {
			return reflect.Value{}, fmt.Errorf("missing option %s with type %v", opt.Name, opt.Type)
		}
		if py.cType != opt.Type {
			return reflect.Value{}, fmt.Errorf(`option missmatch in %s: we expect it to be "%v", but discord says it is "%v"`,
				py.fieldName, py.cType, opt.Type)
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
			u, err := s.User(opt.Value.(string))
			if err == nil {
				men.User = u
			}
			r, err := s.State.Role(i.GuildID, opt.Value.(string))
			if err == nil {
				men.Role = r
			}
			v = men
		default:
			return reflect.Value{}, fmt.Errorf("unrecognized ApplicationCommandOptionType %v in %s", opt.Type, py.fieldName)
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

	for _, arg := range cmdSpecialArg {
		fVal := val.FieldByIndex(arg.fieldIndex)
		switch arg.dataType {
		case cmdDataTypeDiskoiPath:
			fVal.Set(reflect.ValueOf(data.Path))
		default:
			return reflect.Value{}, fmt.Errorf("unrecognized specialArgType %v in %s", arg.dataType, arg.fieldName)
		}
	}
	return val, nil
}

func findPyArg(pys []*CommandArgument, name string) *CommandArgument {
	for _, py := range pys {
		if name == py.Name {
			return py
		}
	}
	return nil
}
