package parser

import (
	"diskoi/mentionable"
	"encoding/csv"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"strconv"
	"strings"
)

var (
	rTypeSession         = reflect.TypeOf((*discordgo.Session)(nil))
	rTypeInteractCreate  = reflect.TypeOf((*discordgo.InteractionCreate)(nil))
	rTypeAppCmdOptChoice = reflect.TypeOf([]*discordgo.ApplicationCommandOptionChoice(nil))
	rTypeUnmarshal       = reflect.TypeOf((*Unmarshal)(nil)).Elem() //todo prefix I for interface types
	rTypeChannelType     = reflect.TypeOf((*ChannelType)(nil)).Elem()
	rTypeCommandOptions  = reflect.TypeOf((*CommandOptions)(nil)).Elem()
)

//AnalyzeCmdFn analyzes the function, and returns Data that can be executed
func AnalyzeCmdFn(fn interface{}) (data *Data, error error) {
	data = &Data{
		fn: fn,
	}
	fnArgs, err := analyzeFunction(fn)
	if err != nil {
		return nil, fmt.Errorf("error analyzing function: %w", err)
	}
	data.fnArg = fnArgs

	if len(fnArgs) >= 1 {
		if arg := fnArgs[len(fnArgs)-1]; arg.typ == fnArgumentTypeData {
			data.cmdStruct = arg.reflectTyp
			data.cmdArg, data.cmdSpecialArg, err = analyzeCommandStruct(arg.reflectTyp, []int{})
			if err != nil {
				return nil, fmt.Errorf(`error analyzing command data(%s): %w`, arg.reflectTyp.String(), err)
			}
		}
	}
	return data, nil
}

func analyzeFunction(fn interface{}) ([]*fnArgument, error) {
	typ := reflect.TypeOf(fn)
	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("given type %s(%s) is not type of func", typ.String(), typ.Kind().String())
	}
	if typ.NumOut() != 0 {
		return nil, fmt.Errorf("given function(%s) has %d outputs, expecting 0", signature(fn), typ.NumOut())
	}

	fnArgs := make([]*fnArgument, 0, typ.NumIn())
	for i := 0; i < typ.NumIn(); i++ {
		fna := &fnArgument{}
		at := typ.In(i)
		original := at
		atp := at
		if atp.Kind() != reflect.Ptr {
			atp = reflect.PtrTo(at)
		}
		switch {
		case at == rTypeSession:
			fna.typ = fnArgumentTypeSession
		case at == rTypeInteractCreate:
			fna.typ = fnArgumentTypeInteraction
		case atp.Implements(rTypeUnmarshal):
			if at.Kind() == reflect.Ptr {
				fna.typ = fnArgumentTypeMarshalPtr
				at = at.Elem()
			} else {
				fna.typ = fnArgumentTypeMarshal
			}
			fna.reflectTyp = at
		default:
			if i < typ.NumIn()-1 { //maybe some day data struct won't have to be the last arg
				return nil, fmt.Errorf("unrecognized argument %s(#%d) on function, "+
					"should be *discordgo.Session, *discordgo.InteractionCreate or something that implement diskoi.Unmarshal", original.String(), i)
			}
			if at.Kind() == reflect.Ptr {
				at = at.Elem()
			}
			if at.Kind() != reflect.Struct {
				return nil, fmt.Errorf("unrecognized argument %s(#%d) on function,"+
					"should be a struct", original.String(), i)
			}
			fna.typ = fnArgumentTypeData
			fna.reflectTyp = at
		}
		fnArgs = append(fnArgs, fna)
	}
	return fnArgs, nil
}

func analyzeAutocompleteFunction(fn interface{}, expTyp reflect.Type) ([]*fnArgument, error) {
	//todo consolidate duplicated code
	typ := reflect.TypeOf(fn)
	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("given type %s(%s) is not type of func", typ.String(), typ.Kind().String())
	}
	if typ.NumOut() != 1 {
		return nil, fmt.Errorf("given function(%s) has %d outputs, expecting 1", signature(fn), typ.NumOut())
	}

	if typ.Out(0) != rTypeAppCmdOptChoice {
		return nil, fmt.Errorf(`given function(%s) should output "%s" not %s`,
			signature(fn), rTypeAppCmdOptChoice.String(), typ.Out(1).String())
	}

	fnArgs := make([]*fnArgument, 0, typ.NumIn()) //todo split this into it's own function
	for i := 0; i < typ.NumIn(); i++ {
		fna := &fnArgument{}
		at := typ.In(i)
		original := at
		atp := at
		if atp.Kind() != reflect.Ptr {
			atp = reflect.PtrTo(at)
		}
		switch {
		case at == rTypeSession:
			fna.typ = fnArgumentTypeSession
		case at == rTypeInteractCreate:
			fna.typ = fnArgumentTypeInteraction
		case atp.Implements(rTypeUnmarshal):
			if at.Kind() == reflect.Ptr {
				fna.typ = fnArgumentTypeMarshalPtr
				at = at.Elem()
			} else {
				fna.typ = fnArgumentTypeMarshal
			}
			fna.reflectTyp = at
		default:
			if i < typ.NumIn()-1 { //maybe some day data struct won't have to be the last arg
				return nil, fmt.Errorf("unrecognized argument %s(#%d) on function, "+
					"should be *discordgo.Session, *discordgo.InteractionCreate or something that implement diskoi.Unmarshal", original.String(), i)
			}
			if at.Kind() == reflect.Ptr {
				at = at.Elem()
			}
			if at != expTyp {
				return nil, fmt.Errorf(`unexpected data struct type should be "%s" not "%s"`, expTyp.String(), at.String())
			}
			fna.typ = fnArgumentTypeData
			fna.reflectTyp = at
		}
		fnArgs = append(fnArgs, fna)
	}
	return fnArgs, nil
}

//analyzeCommandStruct analyzes a struct and create slice of CommandArgument and specialArgument
func analyzeCommandStruct(typ reflect.Type, pre []int) ([]*CommandArgument, []*specialArgument, error) {
	cmdArgs := make([]*CommandArgument, 0, typ.NumField())
	spcArgs := make([]*specialArgument, 0, 1)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		pos := append(append(make([]int, 0, len(pre)+1), pre...), i)
		if !f.IsExported() {
			return nil, nil, fmt.Errorf(`unsupported unexported field in "%s.%s"`, typ.String(), f.Name)
		}
		if f.Anonymous {
			if f.Type.Kind() == reflect.Ptr {
				return nil, nil, fmt.Errorf(`unsupported pointered anonymous field in "%s.%s"`, typ.String(), f.Name)
			}
			a, s, err := analyzeCommandStruct(f.Type, pos)
			if err != nil {
				return nil, nil, fmt.Errorf(`in "%s": %w`, typ.String(), err)
			}
			cmdArgs = append(cmdArgs, a...)
			spcArgs = append(spcArgs, s...)
			continue
		}

		py, pys, err := analyzeCommandArgumentField(f)
		if err != nil {
			return nil, nil, fmt.Errorf(`failed parsing struct field on "%s.%s": %w`, typ.String(), f.Name, err)
		}
		if py != nil {
			py.fieldIndex = pos
			cmdArgs = append(cmdArgs, py)
		}
		if pys != nil {
			pys.fieldIndex = pos
			spcArgs = append(spcArgs, pys)
		}

	}
	return cmdArgs, spcArgs, nil
}

const magicTag = "diskoi"

//analyzeCommandArgumentField analyze a command struct field and return CommandArgument or specialArgument
func analyzeCommandArgumentField(f reflect.StructField) (*CommandArgument, *specialArgument, error) {
	tag, ok := f.Tag.Lookup(magicTag)

	arg := &CommandArgument{
		fieldName: f.Name,
		Name:      strings.ToLower(f.Name),
	}

	if ok {
		r := csv.NewReader(strings.NewReader(tag))
		r.Comment = 0
		r.FieldsPerRecord = -1
		r.TrimLeadingSpace = true

		allEntries, err := r.ReadAll()
		if err != nil {
			return nil, nil, fmt.Errorf(`error parsing tag: %s`, err.Error())
		}
		for _, subEntry := range allEntries {
			for _, ent := range subEntry {
				key, value := splitTxt(ent)
				switch key {
				case "name":
					arg.Name = value
				case "description":
					arg.Description = value
				case "required":
					if len(value) == 0 {
						arg.Required = true
					} else {
						b, err := strconv.ParseBool(value)
						if err != nil {
							arg.Required = true
						} else {
							arg.Required = b
						}
					}
				case "special":
					sp := &specialArgument{}
					switch value {
					case "path":
						sp.dataType = cmdDataTypeDiskoiPath

						if f.Type.Kind() != reflect.Slice || f.Type.Elem().Kind() != reflect.String {
							return nil, nil, fmt.Errorf(`invalid reciever type "%s" on special:path tag`, f.Type.String())
						}
					default:
						return nil, nil, fmt.Errorf("unrecognized special tag with value \"%s\"", value)
					}
					return nil, sp, nil
				default:
					return nil, nil, fmt.Errorf("unrecognized tag \"%s\" with value \"%s\"", key, value)
				}
			}
		}
	}

	if f.Type.Implements(rTypeChannelType) {
		v := reflect.New(f.Type)
		ch := v.Interface().(ChannelType)
		arg.ChannelTypes = ch.DiskoiChannelTypes()
	}
	if f.Type.Implements(rTypeCommandOptions) {
		v := reflect.New(f.Type)
		ch := v.Interface().(CommandOptions)
		arg.Choices = ch.DiskoiCommandOptions()
	}

	elmT := f.Type
	if elmT.Kind() == reflect.Ptr {
		elmT = f.Type.Elem()
	}
	switch elmT.Kind() {
	case reflect.String:
		arg.cType = discordgo.ApplicationCommandOptionString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		arg.cType = discordgo.ApplicationCommandOptionInteger
	case reflect.Bool:
		arg.cType = discordgo.ApplicationCommandOptionBoolean
	case reflect.Float32:
	case reflect.Float64:
		arg.cType = 10 //type doubles fixme get constant from discord go
	case reflect.Struct:
		switch {
		case elmT == reflect.TypeOf(discordgo.Channel{}):
			arg.cType = discordgo.ApplicationCommandOptionChannel
		case elmT == reflect.TypeOf(discordgo.User{}):
			arg.cType = discordgo.ApplicationCommandOptionUser
		case elmT == reflect.TypeOf(discordgo.Role{}):
			arg.cType = discordgo.ApplicationCommandOptionRole
		case elmT == reflect.TypeOf(mentionable.Mentionable{}):
			arg.cType = discordgo.ApplicationCommandOptionMentionable
		default:
			return nil, nil, fmt.Errorf(`unrecognized struct "%s"`, f.Type.String())
		}
	default:
		return nil, nil, fmt.Errorf(`unsupported kind "%s"`, f.Type.String())
	}
	return arg, nil, nil
}

func splitTxt(str string) (string, string) {
	split := strings.SplitN(str, ":", 2)
	if len(split) >= 2 {
		return split[0], split[1]
	}
	return split[0], ""
}
