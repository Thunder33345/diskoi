package diskoi

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
)

type CommandParsingError struct {
	err error
}

func (e CommandParsingError) Error() string {
	return "command parsing error: " + e.err.Error()
}

func (e *CommandParsingError) Unwrap() error {
	return e.err
}

type MissingOptionsError struct {
	path []string
}

func (e MissingOptionsError) Error() string {
	return "missing options(possible api de-sync?): expecting options given for command group, none given for" + errPath(e.path)
}

type NonCommandOptionTypeError struct {
	ty   discordgo.ApplicationCommandOptionType
	path []string
}

func (e NonCommandOptionTypeError) Error() string {
	return "non command option type(possible api de-sync?): expecting \"SubCommand\" or \"SubCommandGroup\" command option type" +
		" but received \"" + e.ty.String() + "\" for" + errPath(e.path)
}

type MissingSubcommandError struct {
	name    string
	path    []string
	isGroup bool
}

func (e MissingSubcommandError) Error() string {
	if e.isGroup {
		return "missing subcommand group: group \"" + e.name + "\" not found on" + errPath(e.path)
	}
	return "missing subcommand: subcommand \"" + e.name + "\" not found on" + errPath(e.path)
}

type CommandExecutionError struct {
	name string
	err  error
}

func (e CommandExecutionError) Error() string {
	return fmt.Sprintf(`error executing command for "%s": %v`, e.name, e.err)
}

func (e CommandExecutionError) Unwrap() error {
	return e.err
}

type AutocompleteExecutionError struct {
	name string
	err  error
}

func (e AutocompleteExecutionError) Error() string {
	return fmt.Sprintf(`error executing autocomplete for "%s": %v`, e.name, e.err)
}

func (e AutocompleteExecutionError) Unwrap() error {
	return e.err
}

type MissingBindingsError struct {
	name string
}

func (e MissingBindingsError) Error() string {
	return "missing bindings for " + e.name
}

type DiscordAPIError struct {
	err error
}

func (e DiscordAPIError) Error() string {
	return fmt.Sprintf(`discord api error: %v`, e.err)
}

func (e DiscordAPIError) Unwrap() error {
	return e.err
}

func errPath(path []string) string {
	return "/" + strings.Join(path, " ")
}
