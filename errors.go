package diskoi

import (
	"fmt"
	"strings"
)

//CommandParsingError indicates error is originated from command parsing, lookup or reconstructing
type CommandParsingError struct {
	err error
}

func (e CommandParsingError) Error() string {
	return fmt.Sprintf("command parsing error: %v", e.err)
}

func (e *CommandParsingError) Unwrap() error {
	return e.err
}

//CommandExecutionError indicates error is originated from executing a command function
type CommandExecutionError struct {
	name string
	err  error
}

func (e CommandExecutionError) Error() string {
	return fmt.Sprintf(`executing command "%s": %v`, e.name, e.err)
}

func (e CommandExecutionError) Unwrap() error {
	return e.err
}

//CommandMiddlewareExecutionError indicates error is originated from executing a middleware between command function
type CommandMiddlewareExecutionError struct {
	name string
	err  error
}

func (e CommandMiddlewareExecutionError) Error() string {
	return fmt.Sprintf(`executing command middleware "%s": %v`, e.name, e.err)
}

func (e CommandMiddlewareExecutionError) Unwrap() error {
	return e.err
}

//AutocompleteExecutionError indicates the error comes from executing an autocomplete handler
type AutocompleteExecutionError struct {
	name string
	err  error
}

func (e AutocompleteExecutionError) Error() string {
	return fmt.Sprintf(`executing autocomplete "%s": %v`, e.name, e.err)
}

func (e AutocompleteExecutionError) Unwrap() error {
	return e.err
}

//DiscordAPIError is used for warping errors produced by discordgo library
type DiscordAPIError struct {
	err error
}

func (e DiscordAPIError) Error() string {
	return fmt.Sprintf(`discord api error: %v`, e.err)
}

func (e DiscordAPIError) Unwrap() error {
	return e.err
}

//DiscordExpectationError is used to wrap text that signifies discord api is returning behaving in unexpected way
type DiscordExpectationError struct {
	err string
}

func (e DiscordExpectationError) Error() string {
	return "discord expectation error: " + e.err
}

func newDiscordExpectationError(err string) error {
	return DiscordExpectationError{err: err}
}

func errPath(path []string) string {
	return "/" + strings.Join(path, " ")
}
