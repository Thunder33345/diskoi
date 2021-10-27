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
		cmd          *CommandArgument
		special      *specialArgument
		wantErr      bool
		wantErrRegex *regexp.Regexp
	}{
		{
			name: "test csv",
			in:   reflect.StructField{Name: "test", Tag: `diskoi:"name:foo,description:foobar,required"`, Type: reflect.TypeOf((*string)(nil))},
			cmd: &CommandArgument{
				fieldName:   "test",
				cType:       discordgo.ApplicationCommandOptionString,
				Name:        "foo",
				Description: "foobar",
				Required:    true,
			},
		}, {
			name: "test csv2",
			in:   reflect.StructField{Name: "test", Tag: `diskoi:"name:foo!!:!,required"`, Type: reflect.TypeOf((*bool)(nil))},
			cmd: &CommandArgument{
				fieldName: "test",
				cType:     discordgo.ApplicationCommandOptionBoolean,
				Name:      "foo!!:!",
				Required:  true,
			},
		}, {
			name: "test csv3",
			in:   reflect.StructField{Name: "test", Tag: `diskoi:"\"name:foobar\",required"`, Type: reflect.TypeOf((*float32)(nil))},
			cmd: &CommandArgument{
				fieldName: "test",
				Name:      "foobar",
				cType:     applicationCommandOptionDouble,
				Required:  true,
			},
		}, {
			name:         "test csv fail",
			in:           reflect.StructField{Name: "test", Tag: `diskoi:"\"name:foo!!:,!,required"`, Type: reflect.TypeOf((*string)(nil))},
			wantErr:      true,
			wantErrRegex: regexp.MustCompile("^error parsing tag:"),
		}, {
			name: "test require false",
			in:   reflect.StructField{Tag: `diskoi:"\"name:foobar\",required:false"`, Type: reflect.TypeOf((*int)(nil))},
			cmd: &CommandArgument{
				Name:     "foobar",
				cType:    discordgo.ApplicationCommandOptionInteger,
				Required: false,
			},
		}, {
			name: "test require true",
			in:   reflect.StructField{Tag: `diskoi:"\"name:foobar\",required:1"`, Type: reflect.TypeOf((*discordgo.Channel)(nil))},
			cmd: &CommandArgument{
				Name:     "foobar",
				cType:    discordgo.ApplicationCommandOptionChannel,
				Required: true,
			},
		}, {
			name: "test require implicit",
			in:   reflect.StructField{Tag: `diskoi:"\"name:foobar\",required:foo"`, Type: reflect.TypeOf((*discordgo.Channel)(nil))},
			cmd: &CommandArgument{
				Name:     "foobar",
				cType:    discordgo.ApplicationCommandOptionChannel,
				Required: true,
			},
		}, {
			name: "test special",
			in:   reflect.StructField{Tag: `diskoi:"special:path"`, Type: reflect.TypeOf(([]string)(nil))},
			special: &specialArgument{
				dataType: cmdDataTypeDiskoiPath,
			},
		}, {
			name:         "test special fail",
			in:           reflect.StructField{Tag: `diskoi:"special:foobar"`, Type: reflect.TypeOf((*string)(nil))},
			wantErr:      true,
			wantErrRegex: regexp.MustCompile("^unrecognized special tag with value"),
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
