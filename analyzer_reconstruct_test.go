package diskoi

import (
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
	"reflect"
	"regexp"
	"testing"
)

func TestReconstructCommandArgument(t *testing.T) {
	cases := []struct {
		name      string
		cmdStruct reflect.Type
		cmdArg    []*commandArgument
		opts      []*discordgo.ApplicationCommandInteractionDataOption

		want    reflect.Value
		wantErr *regexp.Regexp
	}{
		{
			name:      "test",
			cmdStruct: reflect.TypeOf(Reconstruct1{}),
			opts: []*discordgo.ApplicationCommandInteractionDataOption{
				{
					Name:  "string",
					Type:  discordgo.ApplicationCommandOptionString,
					Value: "foobar",
				}, {
					Name:  "int64",
					Type:  discordgo.ApplicationCommandOptionInteger,
					Value: float64(9001),
				}, {
					Name:  "bool",
					Type:  discordgo.ApplicationCommandOptionBoolean,
					Value: true,
				}, {
					Name:  "uint",
					Type:  discordgo.ApplicationCommandOptionInteger,
					Value: float64(10),
				}, {
					Name:  "float64",
					Type:  applicationCommandOptionDouble,
					Value: float64(11.11111),
				}, {
					Name:  "float32",
					Type:  applicationCommandOptionDouble,
					Value: float64(222.2222),
				},
			},
			want: reflect.ValueOf(Reconstruct1{
				String:  "foobar",
				Int64:   9001,
				Bool:    true,
				UInt:    10,
				Float64: 11.11111,
				Float32: 222.2222,
			}),
		}, {
			name:      "err local opt missing",
			cmdStruct: reflect.TypeOf(Reconstruct1{}),
			opts: []*discordgo.ApplicationCommandInteractionDataOption{
				{
					Name:  "404 missing",
					Type:  discordgo.ApplicationCommandOptionString,
					Value: "foobar",
				},
			},
			wantErr: regexp.MustCompile(`^cant find option named ".*?" type of`),
		}, {
			name:      "err remote mismatch",
			cmdStruct: reflect.TypeOf(Reconstruct1{}),
			opts: []*discordgo.ApplicationCommandInteractionDataOption{
				{
					Name:  "string",
					Type:  discordgo.ApplicationCommandOptionInteger,
					Value: 100,
				},
			},
			wantErr: regexp.MustCompile(`^option type mismatch in ".*?": we expect it to be ".*?"`),
		}, {
			name:      "err unrecognized type",
			cmdStruct: reflect.TypeOf(Reconstruct1{}),
			opts: []*discordgo.ApplicationCommandInteractionDataOption{
				{
					Name:  "string",
					Type:  discordgo.ApplicationCommandOptionType(255),
					Value: 100,
				},
			},
			cmdArg: []*commandArgument{{
				fieldIndex: []int{0},
				fieldName:  "string",
				cType:      255,
				Name:       "string",
			}},
			wantErr: regexp.MustCompile(`^unrecognized ApplicationCommandOptionType`),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			var cmdArg []*commandArgument
			if len(tc.cmdArg) >= 1 {
				cmdArg = tc.cmdArg
			} else {
				var err error
				cmdArg, err = analyzeCommandStruct(tc.cmdStruct, nil)
				r.Nil(err)
			}
			value, err := reconstructCommandArgument(tc.cmdStruct, cmdArg, nil, nil, tc.opts)
			if tc.wantErr != nil {
				r.Regexp(tc.wantErr, err)
				return
			} else {
				r.Nil(err)
			}

			if tc.want != reflect.ValueOf(nil) {
				r.EqualValues(tc.want.Interface(), value.Interface())

			} else {
				r.Nil(value)
			}
		})
	}
}

type Reconstruct1 struct {
	String  string
	Int64   int64
	Bool    bool
	UInt    uint
	Float64 float64
	Float32 float32
}
