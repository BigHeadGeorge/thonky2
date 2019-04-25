package main

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"sync"

	spreadsheet "gopkg.in/Iwark/spreadsheet.v2"
)

type Container struct {
	Cells *[7][6]*spreadsheet.Cell
}

func (c *Container) Values() [7][6]string {
	var values [7][6]string
	for i, row := range c.Cells {
		for j, cell := range row {
			values[i][j] = cell.Value
		}
	}
	return values
}

type Player struct {
	Name string
	Role string
	Container
}

type Week struct {
	Date string
	Days *[7]string
	Container
}

func (w *Week) ActivitiesOn(day int) [6]string {
	return w.Values()[day]
}

func (p *Player) Availability() [7][6]string {
	return p.Values()
}

func (p *Player) AvailabilityOn(day int) [6]string {
	return p.Availability()[day]
}

func (p *Player) AvailabilityAt(day, time, start int) string {
	return p.AvailabilityOn(day)[time-start]
}

func getPlayer(s *spreadsheet.Spreadsheet, name string) (Player, error) {
	sheet, err := s.SheetByTitle(name)
	if err != nil {
		return Player{}, err
	}
	var availability [7][6]*spreadsheet.Cell
	for i := 2; i < 9; i++ {
		for j := 2; j < 8; j++ {
			availability[i-2][j-2] = &sheet.Rows[i][j]
		}
	}

	p := Player{Name: name}
	p.Cells = &availability
	return p, nil
}

func GetPlayers(s *spreadsheet.Spreadsheet) []*Player {
	sheet, err := s.SheetByTitle("Team Availability")
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(12)
	pCh := make(chan Player, 12)
	var currentRole string
	var playerCount int
	for i := 3; i < 15; i++ {
		role := sheet.Rows[i][1].Value
		if role != "" && currentRole != role {
			currentRole = role
		}

		name := sheet.Rows[i][2].Value
		if name != "" {
			playerCount++
			go func(name, role string) {
				defer wg.Done()
				player, _ := getPlayer(s, name)
				player.Role = role
				pCh <- player
			}(name, role)
			continue
		}
		wg.Done()
	}

	wg.Wait()
	close(pCh)

	var players []*Player
	for i := 0; i < playerCount; i++ {
		p := <-pCh
		players = append(players, &p)
	}
	return players
}

func GetWeek(s *spreadsheet.Spreadsheet) (*Week, error) {
	sheet, err := s.SheetByTitle("Weekly Schedule")
	if err != nil {
		return &Week{}, err
	}

	date := strings.Split(sheet.Rows[2][1].Value, ", ")[1]

	week := &Week{Date: date}
	var cells [7][6]*spreadsheet.Cell
	var days [7]string
	for i := 2; i < 9; i++ {
		days[i-2] = sheet.Rows[i][1].Value
		for j := 2; j < 8; j++ {
			cells[i-2][j-2] = &sheet.Rows[i][j]
		}
	}
	week.Days = &days
	week.Cells = &cells

	return week, err
}

func saveSheetAttr(c interface{}, attr, sheetID string) error {
	filename := sheetID + "_" + attr + ".json"
	m, err := json.Marshal(c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, m, 0644)
	if err != nil {
		return err
	}
	return nil
}

func loadSheetAttr(attr, sheetID string) (b []byte, err error) {
	b, err = ioutil.ReadFile(sheetID + "_" + attr + ".json")
	return
}
