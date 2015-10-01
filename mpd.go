package mpd

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"reflect"
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
	Time           uint
	Elapsed        string
	Bitrate        uint
	Audio          string
}

type MpdResponse interface{}

func (self *Conn) Status() (status Status, err error) {
	self.WriteLine("status")
	err = self.ReadResponse(&status)
	return status, err
}

func (self *Conn) ReadResponse(status *Status) (err error) {
	line, err := self.ReadLine()
	for ; self.continueReading(line, err); line, err = self.ReadLine() {
		self.parseResponse(status, line)
	}
	return err
}

func (self *Conn) continueReading(line string, err error) bool {
	return err == nil &&
		!strings.HasPrefix(line, "ACK") &&
		!strings.HasPrefix(line, "OK")
}

func (self *Conn) parseResponse(status MpdResponse, line string) (err error) {
	pair := strings.SplitN(line, ":", 2)
	key, val := pair[0], strings.ToLower(strings.TrimSpace(pair[1]))
	fieldName := mapMPDNameToFieldName(key)
	statusElem := reflect.ValueOf(status).Elem()
	field := statusElem.FieldByName(fieldName)

	if field == reflect.Zero(statusElem.Type()) {
		log.Println("Field not found:", field)
	} else {
		switch fmt.Sprintf("%s", field.Type()) {
		case "string":
			field.SetString(val)
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
	default:
		return strings.Title(mpdName)
	}
}
