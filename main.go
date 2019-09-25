package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

type Lyrics struct {
	Title  string `json: "title"`
	Artist string `json: "artist"`
	Lyrics string `json: "lyrics"`
}

type AuddError struct {
	Error_Code    int
	Error_Message string
}

type AuddLyricsRequest struct {
	Status string
	Error  AuddError
	Result []Lyrics
}

func GetLyrics(token, song string) ([]Lyrics, error) {
	resp, err := http.Get("https://api.audd.io/findLyrics/?api_token=" + token + "&q=" + song)
	if err != nil {
		return []Lyrics{}, err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	RespJson := AuddLyricsRequest{}
	json.Unmarshal(body, &RespJson)

	if RespJson.Status != "success" || len(RespJson.Result) < 1 {
		return []Lyrics{}, errors.New("request failed")
	}

	return RespJson.Result, nil
}

func main() {
	conn, _ := net.Dial("tcp", "127.0.0.1:6600")
	mpd := make(chan string)

	go func() {
		for {
			scanner := bufio.NewScanner(conn)
			var r []string
			for scanner.Scan() {
				line := scanner.Text()
				if line == "OK" {
					Changed, State, Title, Album, Artist := "", "", "", "", ""
					for _, s := range r {
						if strings.Fields(s)[0] == "changed:" {
							Changed = s[len("changed: "):len(s)]
						} else if strings.Fields(s)[0] == "state:" {
							State = s[len("state: "):len(s)]
						} else if strings.Fields(s)[0] == "Title:" {
							Title = s[len("Title: "):len(s)]
						} else if strings.Fields(s)[0] == "Album:" {
							Album = s[len("Album: "):len(s)]
						} else if strings.Fields(s)[0] == "Artist:" {
							Artist = s[len("Artist: "):len(s)]
						}
					}
					switch <-mpd {
					case "idle player":
						mpd <- Changed
						break
					case "status":
						mpd <- State
						break
					case "currentsong":
						mpd <- Title
						mpd <- Album
						mpd <- Artist
						break
					}
					r = []string{}
				} else {
					r = append(r, line)
				}
			}
		}
	}()
	fmt.Fprintf(conn, "idle player\n")
	mpd <- "idle player"
	for {
		s := <-mpd
		if s == "player" {
			fmt.Fprintf(conn, "status\n")
			mpd <- "status"

			p := <-mpd
			if p == "play" {
				fmt.Fprintf(conn, "currentsong\n")
				mpd <- "currentsong"

				Title, Album, Artist := <-mpd, <-mpd, <-mpd
				_ = Artist
				_ = Title
				_ = Album

				song := strings.ReplaceAll(string(Artist+"%20"+Title), " ", "%20")

				lry, err := GetLyrics("PUT YOUR AUDD TOKEN", song)
				if err != nil {
					panic(err)
				}
				fmt.Println("\033[2J")

				var buffer string

				arr := strings.Fields(strings.ReplaceAll(lry[0].Lyrics, "\r\n", ""))

				for _, s := range arr {
					if s[0] != '[' && s[len(s)-1] != ']' {
						buffer += s + " "
					} else if s[0] == '[' {
						buffer += "\n/"
					}
				}

				fmt.Println(lry[0].Title)
				fmt.Println(buffer)
			}
			fmt.Fprintf(conn, "idle player\n")
			mpd <- "idle player"
		}
	}
}
