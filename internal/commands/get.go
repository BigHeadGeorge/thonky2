package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bigheadgeorge/thonky2/pkg/command"
	"github.com/bigheadgeorge/thonky2/pkg/schedule"
	"github.com/bigheadgeorge/thonky2/pkg/state"
	"github.com/bwmarrin/discordgo"
)

var (
	timeEmotes = []string{":zero:", ":one:", ":two:", ":three:", ":four:", ":five:", ":six:", ":seven:", ":eight:", ":nine:", ":keycap_ten:", "<:eleven:548022654540578827>", "<:twelve:548220054353870849>"}
)

func init() {
	examples := [][2]string{
		{"!get week", "Show the schedule for this week."},
	}
	command.AddCommand("get", "Get information from the configured spreadsheet.", examples, Get)
}

// Get formats information from a given spreadsheet into a Discord embed.
func Get(s *state.State, m *discordgo.MessageCreate, args []string) (string, error) {
	sched := s.FindSchedule(m.GuildID, m.ChannelID)
	if sched == nil {
		return "", nil
	}
	sheetLink := "https://docs.google.com/spreadsheets/d/" + sched.ID

	var embed *discordgo.MessageEmbed
	if len(args) == 2 {
		switch args[1] {
		case "week":
			log.Println("getting week")
			if sched.Week.Container == nil {
				return "No week schedule, something broke", nil
			}
			embed = formatWeek(s, &sched.Week, sheetLink)
			log.Println("sent week :)")
		case "today":
			log.Println("getting today")
			if sched.Week.Container == nil {
				return "No week schedule, something broke", nil
			} else if sched.Players == nil {
				return "No players, something broke", nil
			}
			embed = formatDay(s, &sched.Week, sched.Players, sheetLink, sched.Week.Today())
		case "unscheduled":
			log.Println("getting unscheduled")
			embed = formatUnscheduled(sched, sheetLink)
		default:
			return fmt.Sprintf("Invalid option for !get: %q", args[1]), nil
		}
	}

	for _, field := range embed.Fields {
		log.Println(*field)
	}
	_, err := s.Session.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		return err.Error(), err
	}
	return "", nil
}

// baseEmbed returns a template embed with the decorative stuff set up all ez
func baseEmbed(title, sheetLink, timezone string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Color: 0x2ecc71,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://cdn.discordapp.com/attachments/437847669839495170/476837854966710282/thonk.png",
		},
		Author: &discordgo.MessageEmbedAuthor{
			Name:    title,
			URL:     sheetLink,
			IconURL: "https://www.clicktime.com/images/web-based/timesheet/integration/googlesheets.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Times shown in " + timezone,
		},
	}
}

// addTimeField adds a field to the given embed with time emotes
func addTimeField(e *discordgo.MessageEmbed, title string, week *schedule.Week) {
	var timeString string
	for i := 0; i < len(week.Container[0])-1; i++ {
		timeString += timeEmotes[week.StartTime+(i*week.BlockLength)] + ", "
	}
	timeString += timeEmotes[week.StartTime+(len(week.Container[0])-1)*week.BlockLength]
	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: title, Value: timeString})
}

// formatWeek formats week information into a Discord embed
func formatWeek(s *state.State, w *schedule.Week, sheetLink string) *discordgo.MessageEmbed {
	embed := baseEmbed("Week of "+w.Date, sheetLink, w.Timezone)
	addTimeField(embed, "Times", w)
	emojiGuild, err := s.Session.Guild("437847669839495168")
	if err != nil {
		return embed
	}

	days := w.Values()
	today := w.Today()
	for i := 0; i < 7; i++ {
		var activityEmojis []string
		currDay := (i + today) % 7
		for j := 0; j < len(days[0]); j++ {
			activity := days[currDay][j]
			if activity == "" || activity == "TBD" {
				activityEmojis = append(activityEmojis, ":grey_question:")
				continue
			}
			emojiName := strings.Replace(strings.ToLower(activity), " ", "_", -1)
			var found bool
			for _, emoji := range emojiGuild.Emojis {
				if emoji.Name == emojiName {
					activityEmojis = append(activityEmojis, emoji.MessageFormat())
					found = true
					break
				}
			}
			if !found {
				activityEmojis = append(activityEmojis, fmt.Sprintf(":regional_indicator_%s:", strings.ToLower(string(activity[0]))))
			}
		}
		var dayName string
		if currDay == today {
			dayName = "**" + w.Days[currDay] + "**"
		} else {
			dayName = w.Days[currDay]
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: dayName, Value: strings.Join(activityEmojis, ", "), Inline: false})
	}

	return embed
}

func roleEmoji(role string) string {
	switch role {
	case "Tanks":
		return ":shield:"
	case "DPS":
		return ":crossed_swords:"
	case "Supports":
		return ":ambulance:"
	case "Coaches":
		return ":books:"
	case "Flex":
		return ":muscle:"
	default:
		return ""
	}
}

func formatDay(s *state.State, w *schedule.Week, p []schedule.Player, sheetLink string, day int) *discordgo.MessageEmbed {
	embed := baseEmbed("Schedule for "+w.Days[day], sheetLink, w.Timezone)
	addTimeField(embed, "Players", w)

	roleAvailability := map[string]*[6]int{}
	for _, player := range p {
		if roleAvailability[player.Role] == nil {
			roleAvailability[player.Role] = &[6]int{}
		}

		var emojis []string
		for i, response := range player.AvailabilityOn(day) {
			var emoji string
			switch response {
			case "Yes":
				roleAvailability[player.Role][i]++
				emoji = ":white_check_mark:"
			case "Maybe":
				emoji = ":grey_question:"
			case "No":
				emoji = ":x:"
			}
			emojis = append(emojis, emoji)
		}
		emojiString := strings.Join(emojis, ", ")

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: roleEmoji(player.Role) + " " + player.Name, Value: emojiString, Inline: false})
	}

	for role, counts := range roleAvailability {
		var availability []string
		for _, count := range counts {
			availability = append(availability, timeEmotes[count])
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: roleEmoji(role) + " " + role, Value: strings.Join(availability, ", "), Inline: false})
	}

	return embed
}

// formatUnscheduled highlights open scrim blocks
func formatUnscheduled(sched *schedule.Schedule, sheetLink string) *discordgo.MessageEmbed {
	embed := baseEmbed("Open Scrims", sheetLink, sched.Week.Timezone)
	addTimeField(embed, "Times", &sched.Week)

	today := sched.Week.Today()
	activities := sched.Week.Values()
	for i := 0; i < 7; i++ {
		currDay := (i + today) % 7
		var open string
		for j, activity := range activities[currDay] {
			if activity == "Scrim" && sched.Week.Container[currDay][j].Note == "" {
				open += ":regional_indicator_o:"
			} else {
				open += ":black_large_square:"
			}
			if j < 5 {
				open += ", "
			}
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: sched.Week.Days[currDay], Value: open, Inline: false})
	}
	return embed
}
