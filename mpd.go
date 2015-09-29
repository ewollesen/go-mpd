package mpd

import (
	"bufio"
	"fmt"
	"log"
	"net"
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
}

func (self *Conn) Status() (status Status, err error) {
	self.WriteLine("status")
	err = self.ReadResponse(&status)
	return status, err
}

func (self *Conn) ReadResponse(status *Status) (err error) {
	for line, err := self.ReadLine(); self.continueReading(line, err); line, err = self.ReadLine() {
		z := strings.SplitN(line, ":", 2)
		key, val := z[0], strings.TrimSpace(z[1])
		switch key {
		case "volume":
			status.Volume, err = strconv.Atoi(val)
			if err != nil {
				log.Fatalln("Error parsing volume", err)
				return err
			}
		case "repeat":
			status.Repeat, err = strconv.ParseBool(val)
			if err != nil {
				return err
			}
		case "random":
			status.Random, err = strconv.ParseBool(val)
			if err != nil {
				return err
			}
		case "single":
			status.Single, err = strconv.ParseBool(val)
			if err != nil {
				return err
			}
		case "consume":
			status.Consume, err = strconv.ParseBool(val)
			if err != nil {
				return err
			}
		case "playlist":
			x, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return err
			}
			status.Playlist = uint(x)
		case "playlistlength":
			x, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return err
			}
			status.PlaylistLength = uint(x)
		case "mixrampdb":
			x, err := strconv.ParseFloat(val, 10)
			if err != nil {
				return err
			}
			status.MixRampDB = float32(x)
		case "state":
			// TODO: validation
			status.State = val
		case "song":
			i, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return err
			}
			status.Song = uint(i)
		case "songid":
			i, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return err
			}
			status.SongId = uint(i)
		case "nextsong":
			i, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return err
			}
			status.NextSong = uint(i)
		case "nextsongid":
			i, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return err
			}
			status.NextSongId = uint(i)
		default:
			log.Println("Unexpected key:", key)
		}
	}
	return err
}

func (self *Conn) continueReading(line string, err error) bool {
	return err == nil &&
		!strings.HasPrefix(line, "ACK") &&
		!strings.HasPrefix(line, "OK")
}
