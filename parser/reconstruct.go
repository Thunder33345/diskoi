package parser

import (
	"diskoi"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
)

func reconstructFunctionArgs(fnArg []*fnArgument, s *discordgo.Session, i *discordgo.InteractionCreate,
	o []*discordgo.ApplicationCommandInteractionDataOption) ([]reflect.Value, error) {
	values := make([]reflect.Value, 0, len(fnArg))
	for _, arg := range fnArg {
		switch arg.typ {
		case fnArgumentTypeSession:
			values = append(values, reflect.ValueOf(s))
		case fnArgumentTypeInteraction:
			values = append(values, reflect.ValueOf(i))
		case fnArgumentTypeMarshal:
			mt := reflect.New(arg.reflectTyp)
			m := mt.Interface().(Unmarshal)
			err := m.UnmarshalDiskoi(s, i, o)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("error unmarshalling %s: %s", arg.reflectTyp.Name(), err.Error()))
			}
			values = append(values, reflect.ValueOf(m))
		default:
			return nil, errors.New(fmt.Sprintf("unrecognized argument type #%d (%s)", uint(arg.typ), arg.typ.String()))
		}
	}
	return values, nil
}
func reconstructPayload(d *Data, s *discordgo.Session, i *discordgo.InteractionCreate,
	opts []*discordgo.ApplicationCommandInteractionDataOption, data DiskoiData) (reflect.Value, error) {
	val := reflect.New(d.pyTy).Elem()
	for _, opt := range opts {
		py := findPyArg(d.pyArg, opt.Name)
		if py == nil {
			return reflect.Value{}, errors.New(fmt.Sprintf("missing option %s with type %v", opt.Name, opt.Type))
		}
		if py.cType != opt.Type {
			return reflect.Value{}, errors.New(fmt.Sprintf(`option missmatch in %s: we expect it to be "%v", but discord says it is "%v"`,
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
			men := &diskoi.Mentionable{}
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
			return reflect.Value{}, errors.New(fmt.Sprintf("unrecognized ApplicationCommandOptionType %v in %s", opt.Type, py.fieldName))
		}
		recVal := reflect.ValueOf(v)
		if fVal.Kind() != reflect.Ptr {
			recVal = recVal.Elem()
		}
		if fVal.Kind() != recVal.Kind() {
			if recVal.CanConvert(fVal.Type()) {
				recVal = recVal.Convert(fVal.Type())
			} else {
				return reflect.Value{}, errors.New(fmt.Sprintf(`cant convert %s(%v) into %s(%v)`,
					recVal.Type().Name(), recVal.Type().Kind(), fVal.Type().Name(), fVal.Type().Kind()))
			}
		}
		fVal.Set(recVal)
	}

	for _, arg := range d.pysArg {
		fVal := val.FieldByIndex(arg.fieldIndex)
		switch arg.dataType {
		case cmdDataTypeDiskoiPath:
			fVal.Set(reflect.ValueOf(data.path))
		default:
			return reflect.Value{}, errors.New(fmt.Sprintf("unrecognized specialArgType %v in %s", arg.dataType, arg.fieldName))
		}
	}
	return val, nil
}

func findPyArg(pys []*PayloadArgument, name string) *PayloadArgument {
	for _, py := range pys {
		if name == py.Name {
			return py
		}
	}
	return nil
}
