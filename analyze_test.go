package diskoi

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
	"github.com/thunder33345/diskoi/interaction"
	"reflect"
	"regexp"
	"testing"
)

func TestAnalyzeCmdFn(t *testing.T) {
	cases := []struct {
		name       string
		fn         interface{}
		wantType   reflect.Type
		wantFnArgs []fnArgument
		wantArgs   []commandArgument
		wantErr    *regexp.Regexp
	}{
		{
			name: "process",
			fn: func(s *discordgo.Session, i *discordgo.InteractionCreate, h interaction.Interaction, et EmbeddableTest) {
			},
			wantType: reflect.TypeOf(EmbeddableTest{}),
			wantFnArgs: []fnArgument{{typ: fnArgumentTypeSession}, {typ: fnArgumentTypeInteraction},
				{
					typ:        fnArgumentTypeMarshal,
					reflectTyp: reflect.TypeOf(interaction.Interaction{}),
				},
				{
					typ:        fnArgumentTypeData,
					reflectTyp: reflect.TypeOf(EmbeddableTest{}),
				},
			},
			wantArgs: []commandArgument{
				{
					fieldIndex: []int{0},
					fieldName:  "Test",
					cType:      discordgo.ApplicationCommandOptionString,
				}, {
					fieldIndex: []int{1},
					fieldName:  "Test2",
					cType:      discordgo.ApplicationCommandOptionInteger,
				}, {
					fieldIndex: []int{2, 0},
					fieldName:  "FooChannel",
					cType:      discordgo.ApplicationCommandOptionChannel,
				}, {
					fieldIndex: []int{2, 1},
					fieldName:  "BarUser",
					cType:      discordgo.ApplicationCommandOptionUser,
				}, {
					fieldIndex: []int{2, 2, 0},
					fieldName:  "FarRole",
					cType:      discordgo.ApplicationCommandOptionRole,
				}, {
					fieldIndex: []int{2, 2, 1},
					fieldName:  "RafInt",
					cType:      discordgo.ApplicationCommandOptionInteger,
				},
			},
		}, {
			name:    "err non func",
			fn:      "foo",
			wantErr: regexp.MustCompile("^given type .*?\\) is not type of func"),
		}, {
			name:    "err unexpected output",
			fn:      func() string { return "" },
			wantErr: regexp.MustCompile("^given function.*?\\) has .*? outputs, expecting 0"),
		}, {
			name:    "err in analyzing fn",
			fn:      func(fail complex64) {},
			wantErr: regexp.MustCompile("^analyzing function: unrecognized data struct argument"),
		}, {
			name:    "err in analyzing cmd data",
			fn:      func(fail EmbeddableFail) {},
			wantErr: regexp.MustCompile("^analyzing command data.*?\\): analyzing field"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			fnArgs, dType, cmdArg, err := analyzeCmdFn(tc.fn)
			if tc.wantErr != nil {
				r.Regexp(tc.wantErr, err)
			} else {
				r.Nil(err)
			}
			r.Equal(tc.wantType, dType)
			if len(tc.wantFnArgs) != 0 {
				for i, arg := range tc.wantFnArgs {
					got := fnArgs[i]
					r.Equal(arg.typ, got.typ, fmt.Sprintf("expected fnArgumentType to be the same on: #%d", i))
					r.Equal(arg.reflectTyp, got.reflectTyp, fmt.Sprintf("expected reflect type to be the same on: #%d", i))
				}
			} else {
				r.Empty(fnArgs)
			}

			if tc.wantArgs != nil {
				for i, wArg := range tc.wantArgs {
					gArg := cmdArg[i]
					r.Equal(wArg.fieldIndex, gArg.fieldIndex)
					r.Equal(wArg.fieldName, gArg.fieldName)
					r.Equal(wArg.cType, gArg.cType)
				}
			} else {
				r.Empty(cmdArg)
			}
		})
	}
}

func TestAnalyzeAutocompleteFunction(t *testing.T) {
	cases := []struct {
		name     string
		fn       interface{}
		expected reflect.Type

		wantArg []fnArgument
		wantErr *regexp.Regexp
	}{
		{
			name: "simple",
			fn: func(s *discordgo.Session, i *discordgo.InteractionCreate, h interaction.Interaction, e Embeddable1,
			) []*discordgo.ApplicationCommandOptionChoice {
				panic("this should not be called")
			},
			expected: reflect.TypeOf(Embeddable1{}),
			wantArg: []fnArgument{{typ: fnArgumentTypeSession}, {typ: fnArgumentTypeInteraction},
				{
					typ:        fnArgumentTypeMarshal,
					reflectTyp: reflect.TypeOf(interaction.Interaction{}),
				},
				{
					typ:        fnArgumentTypeData,
					reflectTyp: reflect.TypeOf(Embeddable1{}),
				},
			},
			wantErr: nil,
		}, {
			name:    "err non func",
			fn:      "foo",
			wantErr: regexp.MustCompile("^given type .*?\\) is not type of func"),
		}, {
			name:    "err no output",
			fn:      func() {},
			wantErr: regexp.MustCompile("^given function.*?\\) has .*? outputs, expecting 1"),
		}, {
			name:    "err wrong output",
			fn:      func() string { return "" },
			wantErr: regexp.MustCompile(`^given function.*?\) should output ".*?" not ".*?"`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			args, err := analyzeAutocompleteFunction(tc.fn, tc.expected)
			if tc.wantErr != nil {
				r.Regexp(tc.wantErr, err)
			} else {
				r.Nil(err)
			}
			if len(tc.wantArg) != 0 {
				for i, arg := range tc.wantArg {
					got := args[i]
					r.Equal(arg.typ, got.typ, fmt.Sprintf("expected fnArgumentType to be the same on: #%d", i))
					r.Equal(arg.reflectTyp, got.reflectTyp, fmt.Sprintf("expected reflect type to be the same on: #%d", i))
				}
			} else {
				r.Empty(args)
			}
		})
	}
}

func TestAnalyzeFunctionArgument(t *testing.T) {
	cases := []struct {
		name     string
		fn       reflect.Type
		expected reflect.Type

		wantArg []fnArgument
		wantErr *regexp.Regexp
	}{
		{
			name: "simple",
			fn: reflect.TypeOf(func(s *discordgo.Session, i *discordgo.InteractionCreate, h interaction.Interaction, e Embeddable1) {
			}),
			expected: reflect.TypeOf(Embeddable1{}),
			wantArg: []fnArgument{{typ: fnArgumentTypeSession}, {typ: fnArgumentTypeInteraction},
				{
					typ:        fnArgumentTypeMarshal,
					reflectTyp: reflect.TypeOf(interaction.Interaction{}),
				},
				{
					typ:        fnArgumentTypeData,
					reflectTyp: reflect.TypeOf(Embeddable1{}),
				},
			},
		}, {
			name: "extra",
			fn: reflect.TypeOf(func(s *discordgo.Session, i *discordgo.InteractionCreate, ctx context.Context, m *MetaArgument, e Embeddable1) {
			}),
			expected: reflect.TypeOf(Embeddable1{}),
			wantArg: []fnArgument{{typ: fnArgumentTypeSession}, {typ: fnArgumentTypeInteraction}, {typ: fnArgumentTypeContext}, {typ: fnArgumentTypeMeta},
				{
					typ:        fnArgumentTypeData,
					reflectTyp: reflect.TypeOf(Embeddable1{}),
				},
			},
		}, {
			name: "missing meta ptr",
			fn: reflect.TypeOf(func(m MetaArgument, e Embeddable1) {
			}),
			expected: reflect.TypeOf(Embeddable1{}),
			wantErr:  regexp.MustCompile(`^unrecognized argument diskoi\.MetaArgument`),
		}, {
			name: "duplicated",
			fn: reflect.TypeOf(func(ctx1 context.Context, s *discordgo.Session, i *discordgo.InteractionCreate,
				ctx context.Context, m *MetaArgument, m2 *MetaArgument, e Embeddable2) {
			}),
			expected: reflect.TypeOf(Embeddable2{}),
			wantArg: []fnArgument{{typ: fnArgumentTypeContext}, {typ: fnArgumentTypeSession}, {typ: fnArgumentTypeInteraction},
				{typ: fnArgumentTypeContext}, {typ: fnArgumentTypeMeta}, {typ: fnArgumentTypeMeta},
				{
					typ:        fnArgumentTypeData,
					reflectTyp: reflect.TypeOf(Embeddable2{}),
				},
			},
		}, {
			name: "ptr marshal",
			fn: reflect.TypeOf(func(h *interaction.Interaction, e *Embeddable1) {
			}),
			expected: reflect.TypeOf(Embeddable1{}),
			wantArg: []fnArgument{
				{
					typ:        fnArgumentTypeMarshalPtr,
					reflectTyp: reflect.TypeOf(interaction.Interaction{}),
				},
				{
					typ:        fnArgumentTypeData,
					reflectTyp: reflect.TypeOf(Embeddable1{}),
				},
			},
		}, {
			name:    "err non func",
			fn:      reflect.TypeOf("foo"),
			wantErr: regexp.MustCompile("^given type .*?\\) is not type of func"),
		}, {
			name:    "err command data out of order",
			fn:      reflect.TypeOf(func(e Embeddable2, s *discordgo.Session) {}),
			wantErr: regexp.MustCompile("^unrecognized argument .*?\\) on function, should be"),
		}, {
			name:    "err command data not struct",
			fn:      reflect.TypeOf(func(s *discordgo.Session, c complex64) {}),
			wantErr: regexp.MustCompile("^unrecognized data struct argument"),
		}, {
			name:     "err mismatch expectation",
			fn:       reflect.TypeOf(func(s *discordgo.Session, e Embeddable2) {}),
			expected: reflect.TypeOf(Embeddable1{}),
			wantErr:  regexp.MustCompile(`^unexpected data struct type should be .*`),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			args, err := analyzeFunctionArgument(tc.fn, tc.expected)
			if tc.wantErr != nil {
				r.Regexp(tc.wantErr, err)
			} else {
				r.Nil(err)
			}
			if len(tc.wantArg) != 0 {
				r.Equal(len(tc.wantArg), len(args), "results and want argument length should be equal")
				for i, arg := range tc.wantArg {
					got := args[i]
					r.Equal(arg.typ, got.typ, fmt.Sprintf("expected fnArgumentType to be the same on: #%d", i))
					r.Equal(arg.reflectTyp, got.reflectTyp, fmt.Sprintf("expected reflect type to be the same on: #%d", i))
				}
			} else {
				r.Empty(args)
			}
		})
	}
}

func TestAnalyzeCommandStruct(t *testing.T) {
	cases := []struct {
		name string
		typ  reflect.Type

		wantCmdArg []commandArgument
		errRegex   *regexp.Regexp
	}{
		{
			name: "general test",
			typ:  reflect.TypeOf(EmbeddableTest{}),
			wantCmdArg: []commandArgument{
				{
					fieldIndex: []int{0},
					fieldName:  "Test",
					cType:      discordgo.ApplicationCommandOptionString,
				}, {
					fieldIndex: []int{1},
					fieldName:  "Test2",
					cType:      discordgo.ApplicationCommandOptionInteger,
				}, {
					fieldIndex: []int{2, 0},
					fieldName:  "FooChannel",
					cType:      discordgo.ApplicationCommandOptionChannel,
				}, {
					fieldIndex: []int{2, 1},
					fieldName:  "BarUser",
					cType:      discordgo.ApplicationCommandOptionUser,
				}, {
					fieldIndex: []int{2, 2, 0},
					fieldName:  "FarRole",
					cType:      discordgo.ApplicationCommandOptionRole,
				}, {
					fieldIndex: []int{2, 2, 1},
					fieldName:  "RafInt",
					cType:      discordgo.ApplicationCommandOptionInteger,
				},
			},
		}, {
			name: "err test unexported",
			typ: reflect.TypeOf(struct {
				foo int
			}{}),
			errRegex: regexp.MustCompile(`^unsupported unexported field in ".*?\.foo"`),
		}, {
			name: "err test anon ptr",
			typ: reflect.TypeOf(struct {
				*Embeddable1
			}{}),
			errRegex: regexp.MustCompile(`^unsupported anonymous field with pointer in ".*?"`),
		}, {
			name: "err test nested",
			typ: reflect.TypeOf(struct {
				EmbeddableFail
			}{}),
			errRegex: regexp.MustCompile(`^in ".*?": analyzing field "diskoi.EmbeddableFail.FooComplex": unsupported kind "complex64"`),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			cmd, err := analyzeCommandStruct(tc.typ, []int{})
			if tc.errRegex != nil {
				r.Regexp(tc.errRegex, err)
			} else {
				r.Nil(err)
			}
			if tc.wantCmdArg != nil {
				for i, wArg := range tc.wantCmdArg {
					gArg := cmd[i]
					r.Equal(wArg.fieldIndex, gArg.fieldIndex)
					r.Equal(wArg.fieldName, gArg.fieldName)
					r.Equal(wArg.cType, gArg.cType)
				}
			} else {
				r.Empty(cmd)
			}
		})
	}
}

func TestAnalyzeCommandArgumentField(t *testing.T) {
	cases := []struct {
		name         string
		in           reflect.StructField
		cmd          *commandArgument
		wantErr      bool
		wantErrRegex *regexp.Regexp
	}{
		{
			name: "test csv",
			in:   reflect.StructField{Name: "test", Tag: `diskoi:"name:foo,description:foobar,required"`, Type: reflect.TypeOf((*string)(nil))},
			cmd: &commandArgument{
				fieldName:   "test",
				cType:       discordgo.ApplicationCommandOptionString,
				Name:        "foo",
				Description: "foobar",
				Required:    true,
			},
		}, {
			name: "test csv2",
			in:   reflect.StructField{Name: "test", Tag: `diskoi:"name:foo!!:!,required"`, Type: reflect.TypeOf((*bool)(nil))},
			cmd: &commandArgument{
				fieldName: "test",
				cType:     discordgo.ApplicationCommandOptionBoolean,
				Name:      "foo!!:!",
				Required:  true,
			},
		}, {
			name: "test csv3",
			in:   reflect.StructField{Name: "test", Tag: `diskoi:"\"name:foobar\",required"`, Type: reflect.TypeOf((*float32)(nil))},
			cmd: &commandArgument{
				fieldName: "test",
				Name:      "foobar",
				cType:     applicationCommandOptionDouble,
				Required:  true,
			},
		}, {
			name:         "test csv fail",
			in:           reflect.StructField{Name: "test", Tag: `diskoi:"\"name:foo!!:,!,required"`, Type: reflect.TypeOf((*string)(nil))},
			wantErr:      true,
			wantErrRegex: regexp.MustCompile("^parsing tag:"),
		}, {
			name:         "test invalid tag",
			in:           reflect.StructField{Name: "test", Tag: `diskoi:"name:foobar,bar:foo"`, Type: reflect.TypeOf((*string)(nil))},
			wantErr:      true,
			wantErrRegex: regexp.MustCompile("^unrecognized tag \".*?\" with"),
		}, {
			name: "test require false",
			in:   reflect.StructField{Tag: `diskoi:"\"name:foobar\",required:false"`, Type: reflect.TypeOf((*int)(nil))},
			cmd: &commandArgument{
				Name:     "foobar",
				cType:    discordgo.ApplicationCommandOptionInteger,
				Required: false,
			},
		}, {
			name: "test require true",
			in:   reflect.StructField{Tag: `diskoi:"\"name:foobar\",required:1"`, Type: reflect.TypeOf((*discordgo.Channel)(nil))},
			cmd: &commandArgument{
				Name:     "foobar",
				cType:    discordgo.ApplicationCommandOptionChannel,
				Required: true,
			},
		}, {
			name: "test usertype",
			in:   reflect.StructField{Tag: `diskoi:"\"name:foobarBar\",required:1"`, Type: reflect.TypeOf((*discordgo.User)(nil))},
			cmd: &commandArgument{
				Name:     "foobarBar",
				cType:    discordgo.ApplicationCommandOptionUser,
				Required: true,
			},
		}, {
			name: "test role type",
			in:   reflect.StructField{Tag: `diskoi:"\"name:foobarBar\",required:0"`, Type: reflect.TypeOf((*discordgo.Role)(nil))},
			cmd: &commandArgument{
				Name:     "foobarBar",
				cType:    discordgo.ApplicationCommandOptionRole,
				Required: false,
			},
		}, {
			name: "test mentionable",
			in:   reflect.StructField{Tag: `diskoi:"\"name:foobarBar\",required:1"`, Type: reflect.TypeOf((*Mentionable)(nil))},
			cmd: &commandArgument{
				Name:     "foobarBar",
				cType:    discordgo.ApplicationCommandOptionMentionable,
				Required: true,
			},
		}, {
			name:         "test require implicit",
			in:           reflect.StructField{Tag: `diskoi:"\"name:foobar\",required:foo"`, Type: reflect.TypeOf((*discordgo.Channel)(nil))},
			wantErr:      true,
			wantErrRegex: regexp.MustCompile("^converting \".*?\" into bool: "),
		}, {
			name:         "test special unrecognized",
			in:           reflect.StructField{Tag: `diskoi:"special:foobar"`, Type: reflect.TypeOf((*string)(nil))},
			wantErr:      true,
			wantErrRegex: regexp.MustCompile(`^unrecognized tag "special" with value "foobar"`),
		}, {
			name:         "test unrecognizable struct",
			in:           reflect.StructField{Tag: `diskoi:"name:foo"`, Type: reflect.TypeOf((*commandArgument)(nil))},
			wantErr:      true,
			wantErrRegex: regexp.MustCompile("^unrecognized struct \".*?\""),
		}, {
			name:         "test unsupported kind",
			in:           reflect.StructField{Tag: `diskoi:"name:foo"`, Type: reflect.TypeOf((complex)(0, 0))},
			wantErr:      true,
			wantErrRegex: regexp.MustCompile("^unsupported kind \".*?\""),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)

			cmd, err := analyzeCommandArgumentField(tc.in)
			if tc.wantErr {
				r.Error(err)
				if tc.wantErrRegex != nil {
					r.Regexp(tc.wantErrRegex, err.Error())
				}
				return
			}
			r.Nil(err)
			if tc.cmd != nil {
				r.EqualValues(tc.cmd, cmd)
			} else {
				r.Nil(cmd)
			}
		})
	}
}

func TestAnalyzeCommandArgumentFieldInterface(t *testing.T) {
	r := require.New(t)
	field := reflect.StructField{
		Type: reflect.TypeOf(&InterfaceTestingStruct{}),
	}
	arg, err := analyzeCommandArgumentField(field)
	r.Nil(err)
	r.EqualValues([]discordgo.ChannelType{discordgo.ChannelTypeGuildText, discordgo.ChannelTypeDM}, arg.ChannelTypes)
	r.EqualValues([]*discordgo.ApplicationCommandOptionChoice{{Name: "one", Value: 1}, {Name: "two", Value: 2}}, arg.Choices)
}

type InterfaceTestingStruct struct {
}

func (i InterfaceTestingStruct) DiskoiCommandOptions() []*discordgo.ApplicationCommandOptionChoice {
	return []*discordgo.ApplicationCommandOptionChoice{{Name: "one", Value: 1}, {Name: "two", Value: 2}}
}

func (i InterfaceTestingStruct) DiskoiChannelTypes() []discordgo.ChannelType {
	return []discordgo.ChannelType{discordgo.ChannelTypeGuildText, discordgo.ChannelTypeDM}
}

type Embeddable1 struct {
	FooChannel *discordgo.Channel
	BarUser    *discordgo.User
	Embeddable2
}

type Embeddable2 struct {
	FarRole *discordgo.Role
	RafInt  int
}

type EmbeddableFail struct {
	FooComplex complex64
}

type EmbeddableTest struct {
	Test  string
	Test2 int
	Embeddable1
}
