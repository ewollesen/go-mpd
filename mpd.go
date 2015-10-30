package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

type Conn struct {
	Host     string
	Port     int
	Password string
	sock     net.Conn
	reader   *bufio.Reader
}

type MpdResponse interface{}

func NewMPDConn(host string, port int, password string) *Conn {
	return &Conn{Host: host, Port: port, Password: password}
}

func (self *Conn) Connect() {
	addr := fmt.Sprintf("%s:%d", self.Host, self.Port)
	sock, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalln("Error connecting:", err)
	}
	self.sock = sock
	_, err = self.ReadLine()
	if err != nil {
		log.Fatalln("Error reading:", err)
	}
}

func (self *Conn) ReadLine() (line string, err error) {
	if self.reader == nil {
		self.reader = bufio.NewReader(self.sock)
	}
	line, err = self.reader.ReadString('\n')
	log.Printf("<= %s", line)
	return line, err
}

func (self *Conn) WriteLine(line string) (err error) {
	written, err := fmt.Fprintln(self.sock, line)
	if err != nil {
		log.Fatalln("Error:", err)
	}
	if written != len(line)+1 {
		log.Fatalln("Didn't write it all!", written, "<", len(line)+1)
	}
	log.Println("=>", line)
	return err
}

func (self *Conn) Close() (err error) {
	return self.WriteLine("close")
}

type Status struct {
	Volume         int
	Repeat         bool
	Random         bool
	Single         bool
	Consume        bool
	Playlist       uint
	PlaylistLength uint
	MixRampDB      float32
	State          string // TODO: enum?
	Song           uint
	SongId         uint
	NextSong       uint
	NextSongId     uint
	Time           [2]uint
	Elapsed        string
	Bitrate        uint
	Audio          string
}

type Stats struct {
	Uptime     int
	PlayTime   int
	Artists    int
	Albums     int
	Songs      int
	DBPlayTime int
	DBUpdate   int
}

type SongInfo struct {
	File          string
	LastModified  string
	Time          uint
	Title         string
	Artist        string
	Date          uint
	Album         string
	Track         uint
	AlbumArtist   string
	Disc          uint
	Pos           uint
	Id            uint
	MildredSongId string
	Name          string
}

func (self *Conn) Status() (status Status, err error) {
	status = Status{}
	self.WriteLine("status")
	lines, err := self.ReadResponse()
	if err != nil {
		return status, err
	}
	for _, line := range lines {
		err = self.parseResponseLine(&status, line)
		if err != nil {
			return status, err
		}
	}
	return status, nil
}

func (self *Conn) Stats() (stats Stats, err error) {
	stats = Stats{}
	self.WriteLine("stats")
	lines, err := self.ReadResponse()
	if err != nil {
		return stats, err
	}
	for _, line := range lines {
		err = self.parseResponseLine(&stats, line)
		if err != nil {
			return stats, err
		}
	}
	return stats, nil
}

func (self *Conn) CurrentSong() (song SongInfo, err error) {
	song = SongInfo{}
	self.WriteLine("currentsong")
	lines, err := self.ReadResponse()
	if err != nil {
		return song, err
	}
	for _, line := range lines {
		err = self.parseResponseLine(&song, line)
		if err != nil {
			return song, err
		}
	}
	return song, nil
}

func (self *Conn) ReadResponse() (lines []string, err error) {
	line, err := self.ReadLine()
	for ; self.continueReading(line, err); line, err = self.ReadLine() {
		lines = append(lines, line)
	}
	return lines, err
}

func (self *Conn) continueReading(line string, err error) bool {
	return err == nil &&
		!strings.HasPrefix(line, "ACK") &&
		!strings.HasPrefix(line, "OK")
}

func (self *Conn) parseResponseLine(resp MpdResponse, line string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	pair := strings.SplitN(line, ":", 2)
	key, val := pair[0], strings.TrimSpace(pair[1])
	fieldName := mapMPDNameToFieldName(key)
	respElem := reflect.ValueOf(resp).Elem()
	field := respElem.FieldByName(fieldName)

	if !field.IsValid() {
		log.Println("Field not found:", field)
	} else {
		switch fmt.Sprintf("%s", field.Type()) {
		case "string":
			field.SetString(val)
		case "[2]uint":
			pair := strings.SplitN(val, ":", 2)
			for idx, uintStr := range pair {
				uintVal, err := strconv.ParseUint(uintStr, 10, 32)
				if err != nil {
					return err
				}
				field.Index(idx).SetUint(uintVal)
			}
		case "int":
			v, err := strconv.ParseInt(val, 10, 32)
			if err != nil {
				return err
			}
			field.SetInt(v)
		case "uint":
			v, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return err
			}
			field.SetUint(v)
		case "bool":
			v, err := strconv.ParseBool(val)
			if err != nil {
				return err
			}
			field.SetBool(v)
		case "float32":
			v, err := strconv.ParseFloat(val, 10)
			if err != nil {
				return err
			}
			field.SetFloat(v)
		default:
			panic(fmt.Sprint("Unable to parse unexpected type",
				field.Type()))
		}

	}
	return nil
}

func mapMPDNameToFieldName(mpdName string) string {
	switch mpdName {
	case "playlistlength":
		return "PlaylistLength"
	case "songid":
		return "SongId"
	case "nextsongid":
		return "NextSongId"
	case "nextsong":
		return "NextSong"
	case "mixrampdb":
		return "MixRampDB"
	case "playtime":
		return "PlayTime"
	case "db_playtime":
		return "DBPlayTime"
	case "db_update":
		return "DBUpdate"
	case "MILDRED_SONGID":
		return "MildredSongId"
	case "Last-Modified":
		return "LastModified"
	default:
		return strings.Title(mpdName)
	}
}

func (self *Conn) Ping() (err error) {
	self.WriteLine("ping")
	_, err = self.ReadResponse()
	if err != nil {
		return err
	} else {
		return nil
	}

}

func main() {
	mpd := NewMPDConn("mildred", 6600, "")
	mpd.Connect()

	err := mpd.Ping()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	stats, err := mpd.Stats()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	fmt.Printf("Stats: %v\n", stats)

	status, err := mpd.Status()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	fmt.Printf("Status: %v\n", status)

	song, err := mpd.CurrentSong()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	fmt.Printf("Current Song: %v\n", song)
}
