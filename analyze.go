package diskoi

import (
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
	rTypeCommandOptions  = reflect.TypeOf([]*discordgo.ApplicationCommandOptionChoice(nil))
	rTypeIUnmarshal      = reflect.TypeOf((*Unmarshal)(nil)).Elem()
	rTypeIChannelType    = reflect.TypeOf((*ChannelType)(nil)).Elem()
	rTypeICommandOptions = reflect.TypeOf((*CommandOptions)(nil)).Elem()
)

const applicationCommandOptionDouble = 10 //type doubles fixme get constant from discord go

//analyzeCmdFn analyzes a given function, insure it matches expected function signatures for an execution function
//and calls analyzeFunctionArgument to analyze the function arguments
//finally it loops thru arguments to find if a function have a command data struct, if so analyzes it to get the args
//todo support function receiving context and executor
//todo remove diskoi:"special:path" in favor of receiving MetaArgument or Request
func analyzeCmdFn(fn interface{}) ([]*fnArgument, reflect.Type, []*commandArgument, []*specialArgument, error) {
	typ := reflect.TypeOf(fn)
	if typ.Kind() != reflect.Func {
		return nil, nil, nil, nil, fmt.Errorf("given type %s(%s) is not type of func", typ.String(), typ.Kind().String())
	}
	if typ.NumOut() != 0 {
		return nil, nil, nil, nil, fmt.Errorf("given function(%s) has %d outputs, expecting 0", signature(fn), typ.NumOut())
	}
	fnArgs, err := analyzeFunctionArgument(reflect.TypeOf(fn), nil)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("analyzing function: %w", err)
	}
	var cmdStruct reflect.Type
	var cmdArg []*commandArgument
	var specialArg []*specialArgument
	if len(fnArgs) >= 1 {
		if arg := fnArgs[len(fnArgs)-1]; arg.typ == fnArgumentTypeData {
			cmdStruct = arg.reflectTyp
			cmdArg, specialArg, err = analyzeCommandStruct(arg.reflectTyp, []int{})
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf(`analyzing command data(%s): %w`, arg.reflectTyp.String(), err)
			}
		}
	}
	return fnArgs, cmdStruct, cmdArg, specialArg, nil
}

//analyzeCmdFn analyzes a given function, insure it matches expected function signatures for an autocomplete function
//it also takes in an expected data type of the main executor
//it returns analyzeFunctionArgument which returns a list of function arguments
//it does not process the data struct as it can reuse the same analyzed data for the command struct
func analyzeAutocompleteFunction(fn interface{}, expTyp reflect.Type) ([]*fnArgument, error) {
	typ := reflect.TypeOf(fn)
	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("given type %s(%s) is not type of func", typ.String(), typ.Kind().String())
	}
	if typ.NumOut() != 1 {
		return nil, fmt.Errorf("given function(%s) has %d outputs, expecting 1", signature(fn), typ.NumOut())
	}

	if typ.Out(0) != rTypeCommandOptions {
		return nil, fmt.Errorf(`given function(%s) should output "%s" not %s`,
			signature(fn), rTypeCommandOptions.String(), typ.Out(1).String())
	}

	return analyzeFunctionArgument(typ, expTyp)
}

//analyzeFunctionArgument analyzes a function's arguments and return a slice of analyzed arguments
//it takes in expected and if it's not nil it expects the data struct(if exist) to be the same
func analyzeFunctionArgument(typ reflect.Type, expected reflect.Type) ([]*fnArgument, error) {
	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("given type %s(%s) is not type of func", typ.String(), typ.Kind().String())
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
		case atp.Implements(rTypeIUnmarshal):
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
			if expected != nil && at != expected {
				return nil, fmt.Errorf(`unexpected data struct type should be "%s" not "%s"`, expected.String(), at.String())
			}
			fna.typ = fnArgumentTypeData
			fna.reflectTyp = at
		}
		fnArgs = append(fnArgs, fna)
	}
	return fnArgs, nil
}

//analyzeCommandStruct analyzes the fields of given struct and recursively path into embedded structs
//it calls analyzeCommandArgumentField for each field
//and return the total of all encountered commandArgument and specialArgument in one slice
//the typ is a "reflect.typeof" command struct
//the second pre []int is the prefix of current depth, which should be nothing when calling it, and only used internally
func analyzeCommandStruct(typ reflect.Type, pre []int) ([]*commandArgument, []*specialArgument, error) {
	cmdArgs := make([]*commandArgument, 0, typ.NumField())
	spcArgs := make([]*specialArgument, 0, 1)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		pos := append(append(make([]int, 0, len(pre)+1), pre...), i)
		if !f.IsExported() {
			return nil, nil, fmt.Errorf(`unsupported unexported field in "%s.%s"`, typ.String(), f.Name)
		}
		if f.Anonymous {
			if f.Type.Kind() == reflect.Ptr {
				return nil, nil, fmt.Errorf(`unsupported anonymous field with pointer in "%s.%s"`, typ.String(), f.Name)
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
			return nil, nil, fmt.Errorf(`analyzing field "%s.%s": %w`, typ.String(), f.Name, err)
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

//analyzeCommandArgumentField analyze a "reflect.StructField"
//and returns either commandArgument or specialArgument for said field
//this is iteratively called by analyzeCommandStruct for each field discovered inside the command struct
func analyzeCommandArgumentField(f reflect.StructField) (*commandArgument, *specialArgument, error) {
	tag, ok := f.Tag.Lookup(magicTag)

	arg := &commandArgument{
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
			return nil, nil, fmt.Errorf(`parsing tag: %s`, err.Error())
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
							return nil, nil, fmt.Errorf(`converting "%s" into bool: %w`, value, err)
						}
						arg.Required = b
					}
				case "special":
					sp := &specialArgument{
						fieldName: f.Name,
					}
					switch value {
					case "path":
						sp.dataType = cmdDataTypeDiskoiPath

						if f.Type.Kind() != reflect.Slice || f.Type.Elem().Kind() != reflect.String {
							return nil, nil, fmt.Errorf(`invalid reciever type "%s" on special:path tag expecting []string`, f.Type.String())
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

	if f.Type.Implements(rTypeIChannelType) {
		v := reflect.New(f.Type.Elem()).Elem()
		ch := v.Interface().(ChannelType)
		arg.ChannelTypes = ch.DiskoiChannelTypes()
	}
	if f.Type.Implements(rTypeICommandOptions) {
		v := reflect.New(f.Type.Elem()).Elem()
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
	case reflect.Float32, reflect.Float64:
		arg.cType = applicationCommandOptionDouble
	case reflect.Struct:
		switch {
		case elmT == reflect.TypeOf(discordgo.Channel{}):
			arg.cType = discordgo.ApplicationCommandOptionChannel
		case elmT == reflect.TypeOf(discordgo.User{}):
			arg.cType = discordgo.ApplicationCommandOptionUser
		case elmT == reflect.TypeOf(discordgo.Role{}):
			arg.cType = discordgo.ApplicationCommandOptionRole
		case elmT == reflect.TypeOf(Mentionable{}):
			arg.cType = discordgo.ApplicationCommandOptionMentionable
		default:
			if len(arg.ChannelTypes) == 0 || len(arg.Choices) == 0 {
				return nil, nil, fmt.Errorf(`unrecognized struct "%s"`, f.Type.String())
			}
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
