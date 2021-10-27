package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
	"reflect"
	"regexp"
	"testing"
)

func TestAnalyzeCommandArgumentField(t *testing.T) {
	cases := []struct {
		name         string
		in           reflect.StructField
		cmd          *commandArgument
		special      *specialArgument
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
			name: "test special",
			in:   reflect.StructField{Tag: `diskoi:"special:path"`, Type: reflect.TypeOf(([]string)(nil))},
			special: &specialArgument{
				dataType: cmdDataTypeDiskoiPath,
			},
		}, {
			name:         "test special invalid receiver",
			in:           reflect.StructField{Tag: `diskoi:"special:path"`, Type: reflect.TypeOf(([]int)(nil))},
			wantErr:      true,
			wantErrRegex: regexp.MustCompile("^invalid reciever type \""),
		}, {
			name:         "test special fail",
			in:           reflect.StructField{Tag: `diskoi:"special:foobar"`, Type: reflect.TypeOf((*string)(nil))},
			wantErr:      true,
			wantErrRegex: regexp.MustCompile("^unrecognized special tag with value"),
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

			cmd, special, err := analyzeCommandArgumentField(tc.in)
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
			if tc.special != nil {
				r.EqualValues(tc.special, special)
			} else {
				r.Nil(special)
			}
		})
	}
}

func TestAnalyzeCommandArgumentFieldInterface(t *testing.T) {
	r := require.New(t)
	field := reflect.StructField{
		Type: reflect.TypeOf(&InterfaceTestingStruct{}),
	}
	arg, special, err := analyzeCommandArgumentField(field)
	r.Nil(special)
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
