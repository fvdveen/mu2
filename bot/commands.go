package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/fvdveen/mu2/db"
	"github.com/sirupsen/logrus"
)

// Command is a command in the bot
type Command interface {
	Name() string
	Help() string
	Run(Context, []string)
}

func (b *bot) commandHandler() func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		p := b.Prefix()

		if !strings.HasPrefix(m.Content, p) {
			return
		}

		msg := strings.Split(strings.TrimPrefix(m.Content, p), " ")

		c, err := b.Command(msg[0])
		if err != nil {
			c, err = b.dbCommand(s, m, msg[0])
			if err == db.ErrItemNotFound {
				logrus.WithFields(map[string]interface{}{"type": "handler", "handler": "command"}).Debugf("Command not found: %s", msg[0])
				return
			} else if err != nil {
				logrus.WithFields(map[string]interface{}{"type": "handler", "handler": "command"}).Errorf("Get command: %v", err)
				return
			}
		}

		ctx := b.NewContext(context.Background(), m, s)

		c.Run(ctx, msg[1:])
	}
}

// NewCommand creates a new command
func NewCommand(name string, description string, action func(Context, []string)) Command {
	return &defaultCommand{
		n: name,
		d: description,
		a: action,
	}
}

type defaultCommand struct {
	n string
	d string
	a func(Context, []string)
}

func (c defaultCommand) Name() string {
	return c.n
}

func (c defaultCommand) Help() string {
	return c.d
}

func (c defaultCommand) Run(ctx Context, args []string) {
	c.a(ctx, args)
}

// HelpCommand returns the default help command
func (b *bot) HelpCommand() Command {
	return NewCommand("help", "Sends an help message", func(c Context, _ []string) {
		var msg string
		p := b.Prefix()

		for _, c := range b.Commands() {
			msg = fmt.Sprintf("%s`%s%s` %s\n", msg, p, c.Name(), c.Help())
		}

		if err := c.Send(msg); err != nil {
			logrus.WithFields(map[string]interface{}{"type": "command", "command": "help"}).Errorf("Send message: %v", err)
			return
		}
	})
}

// InfoCommand returns the default help command
func (b *bot) InfoCommand() Command {
	return NewCommand("info", "Sends info about the bot", func(c Context, _ []string) {
		e := &discordgo.MessageEmbed{
			Title:       "Mu2",
			Description: "Info",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Author",
					Value: "Friso van der Veen",
				},
				{
					Name:  "Server count",
					Value: strconv.Itoa(len(c.Session().State.Guilds)),
				},
				{
					Name:  "Invite link",
					Value: fmt.Sprintf("https://discordapp.com/oauth2/authorize?client_id=%s&scope=bot", c.Session().State.User.ID),
				},
				{
					Name:  "Github",
					Value: "https://github.com/fvdveen/mu2",
				},
			},
		}

		if err := c.SendEmbed(e); err != nil {
			logrus.WithFields(map[string]interface{}{"type": "command", "command": "help"}).Errorf("Send embed: %v", err)
			return
		}
	})
}

func dbCommand(i *db.Item) Command {
	return NewCommand("", "", func(c Context, _ []string) {
		if err := c.Send(i.Response); err != nil {
			logrus.WithFields(map[string]interface{}{
				"type":    "command",
				"command": "_learnable",
				"item":    i.Message,
				"guildID": i.GuildID,
			}).Errorf("Send message: %v", err)
			return
		}
	})
}

func (b *bot) learnCommands() []Command {
	return []Command{
		NewCommand("learn", "Teach a message-response command to the bot", func(c Context, args []string) {
			switch args[0] {
			case "learn", "unlearn", "help", "info":
				if err := c.Send("Very funny..."); err != nil {
					logrus.WithFields(map[string]interface{}{"type": "command", "command": "unlearn"}).Errorf("Send message: %v", err)
					return
				}
				return
			}

			g, err := c.Guild()
			if err != nil {
				logrus.WithFields(map[string]interface{}{"type": "command", "command": "learn"}).Errorf("Get guild: %v", err)
				return
			}

			_, err = c.Database().Get(g.ID, args[0])
			if err != nil && err != db.ErrItemNotFound {
				logrus.WithFields(map[string]interface{}{"type": "command", "command": "learn"}).Errorf("Get item: %v", err)
				return
			} else if err == nil {
				if err := c.Send("haha, no"); err != nil {
					logrus.WithFields(map[string]interface{}{"type": "command", "command": "learn"}).Errorf("Send message: %v", "haha, no")
					return
				}
				return
			}

			err = c.Database().New(&db.Item{
				Message:  args[0],
				Response: strings.Join(args[1:], " "),
				GuildID:  g.ID,
			})
			if err != nil {
				logrus.WithFields(map[string]interface{}{"type": "command", "command": "learn"}).Errorf("Store item: %v", err)
				return
			}

			if err := c.Send(fmt.Sprintf("Learned %s successfully", args[0])); err != nil {
				logrus.WithFields(map[string]interface{}{"type": "command", "command": "learn"}).Errorf("Send message: %v", err)
				return
			}
		}),
		NewCommand("unlearn", "Unlearn a message-response command", func(c Context, args []string) {
			switch args[0] {
			case "learn", "unlearn", "help", "info":
				if err := c.Send("Very funny..."); err != nil {
					logrus.WithFields(map[string]interface{}{"type": "command", "command": "unlearn"}).Errorf("Send message: %v", err)
					return
				}
				return
			}

			g, err := c.Guild()
			if err != nil {
				logrus.WithFields(map[string]interface{}{"type": "command", "command": "unlearn"}).Errorf("Get guild: %v", err)
				return
			}

			if err := c.Database().Remove(g.ID, args[0]); err != nil {
				logrus.WithFields(map[string]interface{}{"type": "command", "command": "unlearn"}).Errorf("Get guild: %v", err)
				return
			}

			if err := c.Send(fmt.Sprintf("Unlearned %s successfully", args[0])); err != nil {
				logrus.WithFields(map[string]interface{}{"type": "command", "command": "unlearn"}).Errorf("Send message: %v", err)
				return
			}
		}),
	}
}

func (b *bot) dbCommand(s *discordgo.Session, m *discordgo.MessageCreate, msg string) (Command, error) {
	ch, err := s.State.Channel(m.ChannelID)
	if err != nil {
		ch, err = s.Channel(m.ChannelID)
		if err != nil {
			return nil, err
		}
	}

	i, err := b.db.Get(ch.GuildID, msg)
	if err == db.ErrItemNotFound {
		return nil, db.ErrItemNotFound
	} else if err != nil {
		return nil, err
	}

	return dbCommand(i), nil
}
