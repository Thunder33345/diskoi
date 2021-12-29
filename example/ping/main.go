package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/thunder33345/diskoi"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	BotToken := flag.String("token", "", "Discord bot token")
	GuildID := flag.String("guild", "", "Guild ID for example command")
	flag.Parse()
	if BotToken == nil || GuildID == nil { //requires both flags to be set
		panic("-token and -guild must be set, see -h")
	}

	d := diskoi.NewDiskoi()
	s, err := discordgo.New("Bot " + *BotToken)
	if err != nil {
		panic(err)
	}
	d.RegisterSession(s)

	d.SetErrorHandler(func(_ *discordgo.Session, _ *discordgo.InteractionCreate, cmd diskoi.Command, err error) {
		fmt.Printf(`Error on command "%s": %v`+"\n", cmd.Name(), err)
	})

	//add commands into diskoi, should be done BEFORE Session.Open
	customStop := addCommands(d, *GuildID)

	//register a handler to sync command
	s.AddHandlerOnce(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Ready!")
		//when state is ready call sync commands, this is when commands will be sent to discord
		err = d.SyncCommands()
		if err != nil {
			panic(err)
		}
		fmt.Println("Commands registered!")
	})

	//open to actually start the bot
	err = s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer s.Close()

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		fmt.Println("Interrupt received, stopping bot")
	case msg := <-customStop:
		fmt.Println("Discord stop command received, stopping bot")
		if msg != nil {
			fmt.Printf("Custom stop message recieved: %s\n", *msg)
		}
	}
	s.Close()
}

func addCommands(d *diskoi.Diskoi, guild string) chan *string {
	shutdown := make(chan *string, 1)

	d.AddGuildCommand(guild, diskoi.MustNewExecutor("ping", "Show the latency between this bot and discord",
		func(s *discordgo.Session, i *discordgo.InteractionCreate) error { //simple ping command that dumps some ping metrics
			startTime := time.Now()
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return err
			}
			deferDur := time.Now().Sub(startTime)

			lastSent := startTime.Sub(s.LastHeartbeatSent)
			lastAck := startTime.Sub(s.LastHeartbeatAck)
			_, err = s.InteractionResponseEdit(s.State.User.ID, i.Interaction, &discordgo.WebhookEdit{
				Content: fmt.Sprintf("Heartbeat latency: %v\nLast sent:%v, Last ack: %v\nInteraction latency: %v\n", s.HeartbeatLatency(), lastSent, lastAck, deferDur),
			})
			if err != nil {
				return err
			}
			return nil
		}))

	shutdownCmd := diskoi.MustNewExecutor("shutdown", "Shutdown this bot",
		func(s *discordgo.Session, i *discordgo.InteractionCreate, arg shutdownArgs) error { //uses custom argument struct
			if !arg.Confirm {
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: "You need to confirm you want to shutdown!"},
				})
				return err
			}

			var err error
			if arg.Message != nil { //if ptr is nil, that means it's not set
				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: "Shutting down with custom message..."},
				})
			} else {
				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: "Shutting down..."},
				})
			}
			shutdown <- arg.Message
			return err
		})
	//make sure only users with manage server of said guild can shut the bot down
	_ = shutdownCmd.SetChain(diskoi.NewChain(EnforcePermissions(discordgo.PermissionManageServer, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "You dont have permission to run this command!"},
	})))
	d.AddGuildCommand(guild, shutdownCmd)

	return shutdown
}

//shutdownArgs is a struct for the shutdown command
//the arg is what diskoi parses and tell discord what command arguments we want
//the value parsing is all handled by diskoi
type shutdownArgs struct {
	Confirm bool    `diskoi:"\"description:Are you sure you want to shut the bot down,this will stop the bot?\",required"`
	Message *string `diskoi:"description:Send a shutdown message to the console"`
}

//EnforcePermissions is a middleware that requires permission flags to met before continuing
func EnforcePermissions(flags int64, resp *discordgo.InteractionResponse) diskoi.Chainer {
	return func(next diskoi.Middleware) diskoi.Middleware {
		return func(r diskoi.Request) error {
			if r.Interaction().GuildID == "" {
				err := r.Session().InteractionRespond(r.Interaction().Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: "Run this command in a guild!"},
				})
				//returning an error bubbles up to diskoi.SetErrorHandler, errors should only be returned when unexpected/unhandled
				return err
			}

			if r.Interaction().Member.Permissions&flags != 0 {
				//user has permission for given flags, calling next to proceed
				return next(r)
			}
			//user do not have required flags, displaying error and returning
			err := r.Session().InteractionRespond(r.Interaction().Interaction, resp)
			return err
		}
	}
}
