package parser

import (
	"diskoi/mentionable"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"strconv"
	"strings"
)

var (
	rTypeSession        = reflect.TypeOf((*discordgo.Session)(nil))
	rTypeInteractCreate = reflect.TypeOf((*discordgo.InteractionCreate)(nil))
	rTypeUnmarshal      = reflect.TypeOf((*Unmarshal)(nil)).Elem()
	rTypeChannelType    = reflect.TypeOf((*ChannelType)(nil)).Elem()
	rTypeCommandOptions = reflect.TypeOf((*CommandOptions)(nil)).Elem()
)

func Analyze(fn interface{}) (data *Data, error error) {
	data = &Data{
		fn: fn,
	}
	typ := reflect.TypeOf(fn)
	if typ.Kind() != reflect.Func {
		return nil, errors.New(fmt.Sprintf("given type %s(%s) is not type of func", typ.Name(), typ.Kind().String()))
	}
	if typ.NumOut() != 0 {
		return nil, errors.New(fmt.Sprintf("given function(%s) has %d outputs, expecting 0", signature(fn), typ.NumOut()))
	}

	data.fnArg = make([]*fnArgument, 0, typ.NumIn())
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
			if i < typ.NumIn()-1 {
				return nil, errors.New(fmt.Sprintf("unrecognized argument %s(#%d) on function, "+
					"should be *discordgo.Session, *discordgo.InteractionCreate or something that implement diskoi.Unmarshal", at.Name(), i))
			}
			if at.Kind() == reflect.Ptr {
				at = at.Elem()
			}
			if at.Kind() != reflect.Struct {
				return nil, errors.New(fmt.Sprintf("unrecognized argument %s(#%d) on function,"+
					"should be a struct", at.Name(), i))
			}
			py, pys, err := analyzePayload(at, []int{})
			if err != nil {
				return nil, errors.New(fmt.Sprintf("error parsing command data(%s): %v", at.Name(), err))
			}
			data.pyTy = at
			data.pyArg = py
			data.pysArg = pys
			return data, nil
		}
		data.fnArg = append(data.fnArg, fna)
	}

	return data, nil
}

func analyzePayload(typ reflect.Type, pre []int) ([]*PayloadArgument, []*specialArgument, error) {
	cmdArgs := make([]*PayloadArgument, 0, typ.NumField())
	spcArgs := make([]*specialArgument, 0, 1)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		pos := append(append(make([]int, 0, len(pre)+1), pre...), i)
		if !f.IsExported() {
			return nil, nil, errors.New(fmt.Sprintf(`unsupported unexported field in "%s.%s"`, typ.Name(), f.Name))
		}
		if f.Anonymous {
			if f.Type.Kind() == reflect.Ptr {
				return nil, nil, errors.New(fmt.Sprintf(`unsupported pointered anonymous field in "%s.%s"`, typ.Name(), f.Name))
			}
			a, s, err := analyzePayload(f.Type, pos)
			if err != nil {
				return nil, nil, errors.New(fmt.Sprintf(`in "%s": %s`, typ.Name(), err.Error()))
			}
			cmdArgs = append(cmdArgs, a...)
			spcArgs = append(spcArgs, s...)
			continue
		}

		py, pys, err := analyzePayloadField(f)
		if err != nil {
			return nil, nil, errors.New(fmt.Sprintf(`failed parsing struct field on "%s.%s": %s`, typ.Name(), f.Name, err.Error()))
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

//analyzePayloadField analyze a field of a payload and return cmdBinding for said fields
func analyzePayloadField(f reflect.StructField) (*PayloadArgument, *specialArgument, error) {
	tag, ok := f.Tag.Lookup(magicTag)

	arg := &PayloadArgument{
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
			return nil, nil, errors.New(fmt.Sprintf(`error parsing tag: %s`, err.Error()))
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
							return nil, nil, errors.New(fmt.Sprintf(`invalid reciever type "%s" on special:path tag`, f.Type.String()))
						}
					default:
						return nil, nil, errors.New(fmt.Sprintf("unrecognized special tag with value \"%s\"", value))
					}
					return nil, sp, nil
				default:
					return nil, nil, errors.New(fmt.Sprintf("unrecognized tag \"%s\" with value \"%s\"", key, value))
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
			return nil, nil, errors.New(fmt.Sprintf(`unrecognized struct "%s"`, f.Type.Name()))
		}
	default:
		return nil, nil, errors.New(fmt.Sprintf(`unsupported kind "%s"`, elmT.String()))
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
