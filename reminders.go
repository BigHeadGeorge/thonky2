package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron"
)

// scheduler keeps the Cron scheduler used for reminders from being eaten by the garbage collecter.
// idk if this is actually necessary but whatever :)
var scheduler *cron.Cron

// reminderCheck will check if there's an activity coming up that needs pinging
type reminderCheck struct {
	session *discordgo.Session
}

func (r reminderCheck) Run() {
	for _, info := range guildInfo {
		if info.DocKey.Valid && info.AnnounceChannel.Valid {
			today := time.Now()

			activities := info.Week.ActivitiesOn(Weekday(int(today.Weekday())))
			var done bool
			for i, activity := range activities {
				if done {
					break
				}
				for _, reminder := range info.RemindActivities {
					if activity == reminder {
						done = true
						if i < today.Hour()-3 {
							break
						}

						later := today
						later.Add(time.Duration(time.Hour))
						timeUntil := int(later.Sub(today).Minutes())

						announcement := fmt.Sprintf("%s in %d minutes", activity, timeUntil)
						if info.RoleMention.Valid {
							announcement = info.RoleMention.String + " " + announcement
						}

						r.session.ChannelMessageSend(info.AnnounceChannel.String, announcement)

						announceLog := fmt.Sprintf("send announcement for %q in [%s]", activity, info.GuildID)
						if info.TeamName != "" {
							announceLog += fmt.Sprintf("for %q", info.TeamName)
						}
						log.Println(announceLog)
						break
					}
				}
			}
		}
	}
}

// StartReminders starts checking for reminders 45 and 15 minutes before each hour
func StartReminders(s *discordgo.Session) error {
	scheduler = cron.New()
	err := scheduler.AddJob("0 15 3-8 * * *", reminderCheck{s})
	if err != nil {
		return err
	}
	err = scheduler.AddJob("0 45 3-8 * * *", reminderCheck{s})
	if err != nil {
		return err
	}
	scheduler.Start()
	return nil
}
