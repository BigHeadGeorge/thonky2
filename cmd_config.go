package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"

	"github.com/bigheadgeorge/thonky2/db"
	"github.com/bwmarrin/discordgo"
	"github.com/lib/pq"
)

func init() {
	examples := [][2]string{
		{"!add_team Test #general", "Add a team with the name \"Test\" in #general chat"},
	}
	AddCommand("add_team", "Add a team to the server.", examples, AddTeam)
}

func isChannel(s string) bool {
	match, _ := regexp.MatchString(`<#\d{18}>`, s)
	return match
}

// AddTeam adds a team to a guild
func AddTeam(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	info := guildInfo[m.GuildID]
	if info == nil {
		log.Printf("no info for [%s]\n", m.GuildID)
		s.ChannelMessageSend(m.GuildID, "No info for this guild.")
		return
	}

	if len(args) != 3 {
		if len(args) == 2 {
			s.ChannelMessageSend(m.ChannelID, "Bad amount of args; no channel given!")
		} else {
			s.ChannelMessageSend(m.ChannelID, "Bad amount of args.")
		}
		return
	}
	if !isChannel(args[2]) {
		s.ChannelMessageSend(m.ChannelID, "Invalid channel.")
		return
	}
	channelID := args[2][2 : len(args[2])-1]
	me, err := s.User("@me")
	perms, err := s.State.UserChannelPermissions(me.ID, channelID)
	if err != nil {
		log.Println(err)
		return
	} else if perms&discordgo.PermissionSendMessages == 0 {
		s.ChannelMessageSend(m.ChannelID, "I don't have permission to send messages in that channel. :(")
		return
	}

	handler, err := db.NewHandler()
	if err != nil {
		log.Println(err)
		return
	}
	defer handler.Close()

	if name, err := handler.GetTeamName(args[2][2 : len(args[2])-1]); err == nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Channel already occupied by %q", name))
		return
	}

	config, err := handler.GetGuild("0")
	if err != nil {
		log.Println(err)
		return
	}
	config.GuildID = m.GuildID
	config.TeamName = args[1]
	channelInt, _ := strconv.Atoi(channelID)
	config.Channels = pq.Int64Array([]int64{int64(channelInt)})
	r, err := handler.Query("INSERT INTO teams (server_id, team_name, channels, remind_activities, remind_intervals, update_interval) VALUES ($1, $2, $3, $4, $5, $6)", config.GuildID, config.TeamName, config.Channels, config.RemindActivities, config.RemindIntervals, config.UpdateInterval)
	if err != nil {
		log.Println(err)
		return
	}
	defer r.Close()

	err = info.AddTeam(config)
	if err != nil {
		log.Println(err)
	}
	s.ChannelMessageSend(m.ChannelID, "Added team.")
	log.Printf("added team %q to guild [%s]\n", args[1], m.GuildID)
}