package odscraper

import (
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type PlayerStats struct {
	BTag  string
	SR    int
	Roles map[string]int
}

func (p *PlayerStats) GetMain() string {
	var topRole string
	var topWins int
	for role, wins := range p.Roles {
		if wins > topWins {
			topWins = wins
			topRole = role
		}
	}
	return topRole
}

func parseRole(tokenizer *html.Tokenizer, roles map[string]int) {
	var currentRole string
	for {
		tt := tokenizer.Next()

		if tt == html.EndTagToken {
			t := tokenizer.Token()
			if t.Data == "tr" {
				return
			}
		}

		if tt == html.StartTagToken {
			t := tokenizer.Token()
			if t.Data == "td" && len(t.Attr) > 0 {
				firstVal := t.Attr[0].Val

				if firstVal != "" {
					games, err := strconv.Atoi(firstVal)
					if err == nil {
						roles[currentRole] = games
					}
				}
			} else if t.Data == "a" {
				if t.Attr[1].Val == "color-white" {
					tt = tokenizer.Next()
					currentRole = tokenizer.Token().Data
				}
			}
		}
	}
}

func parsePlayer(r io.Reader) (p PlayerStats) {
	tokenizer := html.NewTokenizer(r)

	var sr string
	p.Roles = map[string]int{
		"Offense": 0,
		"Defense": 0,
		"Support": 0,
		"Tank":    0,
	}

	for {
		tt := tokenizer.Next()

		if tt == html.ErrorToken {
			break
		}

		if tt == html.StartTagToken {
			t := tokenizer.Token()
			if t.Data == "span" && len(t.Attr) == 1 { // check for sr
				if t.Attr[0].Val == "player-skill-rating" {
					tt = tokenizer.Next()
					t = tokenizer.Token()
					if tt == html.TextToken {
						sr = strings.Replace(t.Data, " ", "", -1)
					}
				}
			} else if t.Data == "section" { // check for roles
				for i := 0; i < 2; i++ {
					tt = tokenizer.Next()
				}
				if tt == html.TextToken {
					t = tokenizer.Token()
					if t.Data == "Roles" {
						for {
							tt = tokenizer.Next()

							if tt == html.StartTagToken {
								t = tokenizer.Token()
								if t.Data == "tbody" && len(t.Attr) == 1 {
									for i := 0; i < 4; i++ {
										parseRole(tokenizer, p.Roles)
									}
									break
								}
							}
						}
					}
				}

			}
		}
	}

	if sr == "" {
		p.SR = -1
	} else {
		p.SR, _ = strconv.Atoi(sr)
	}

	return
}

func GetPlayer(btag string) (PlayerStats, error) {
	if match, _ := regexp.MatchString("\\w{1,}#\\d{3,5}", btag); !match {
		return PlayerStats{}, errors.New("invalid btag")
	}

	validTag := strings.Replace(btag, "#", "-", 1)
	resp, err := http.Get(fmt.Sprintf("https://www.overbuff.com/players/pc/%s", validTag))
	if err != nil {
		return PlayerStats{}, err
	}
	defer resp.Body.Close()

	p := parsePlayer(resp.Body)
	p.BTag = btag
	return p, nil
}
