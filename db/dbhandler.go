package db

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/bigheadgeorge/thonky2/schedule"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// NewHandler constructs a new Handler.
func NewHandler() (handler Handler, err error) {
	var b []byte
	b, err = ioutil.ReadFile("config.json")
	if err != nil {
		return
	}

	config := struct {
		User string
		Pw   string
		Host string
	}{}
	err = json.Unmarshal(b, &config)

	connStr := fmt.Sprintf("user=%s password=%s host=%s dbname=thonkydb", config.User, config.Pw, config.Host)
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return
	}
	handler.DB = db
	return
}

// Handler makes grabbing and updating config easier
type Handler struct {
	*sqlx.DB
}

// GetTeamName returns the name of a team in a given channel
func (d *Handler) GetTeamName(channelID string) (string, error) {
	var teamName string
	err := d.Get(&teamName, "SELECT team_name FROM teams WHERE $1 = ANY(channels)", channelID)
	return teamName, err
}

// GetTeams gets the config for each team in a server
func (d *Handler) GetTeams(guildID string) ([]*TeamConfig, error) {
	teams := []*TeamConfig{}
	err := d.Select(&teams, "SELECT * FROM teams WHERE server_id=$1", guildID)
	return teams, err
}

// GetGuild gets the config for a guild
func (d *Handler) GetGuild(guildID string) (*TeamConfig, error) {
	guild := &TeamConfig{}
	err := d.Get(guild, "SELECT * FROM server_config WHERE server_id=$1", guildID)
	return guild, err
}

// ExecJSON runs a query with a JSON representation of v.
func (d *Handler) ExecJSON(query string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = d.Exec(query, b)
	return err
}

// CacheSchedule adds a new schedule to the cache, or updates an existing cache for the schedule.
func (d *Handler) CacheSchedule(s *schedule.Schedule) (err error) {
	r, err := d.Query("SELECT id FROM cache WHERE id = $1", s.ID)
	if err != nil {
		return
	}
	var query string
	update := r.Next()
	if update {
		query = "UPDATE cache SET modified = $1, players = $2, week = $3, activities = $4"
	} else {
		query = "INSERT INTO cache(id, modified, players, week, activities) VALUES($1, $2, $3, $4, $5)"
	}

	var b [2][]byte
	b[0], err = json.Marshal(s.Players)
	if err != nil {
		return
	}
	b[1], err = json.Marshal(s.Week)
	if err != nil {
		return
	}
	activities := pq.StringArray(s.ValidActivities)

	if update {
		_, err = d.Exec(query, s.LastModified, b[0], b[1], activities)
	} else {
		_, err = d.Exec(query, s.ID, s.LastModified, b[0], b[1], activities)
	}
	return
}
