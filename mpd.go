package mpd

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

type Conn struct {
	Host     string
	Port     int
	Password string
	sock     net.Conn
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
	defer sock.Close()
	version, err := self.ReadLine()
	if err != nil {
		log.Fatalln("Error reading:", err)
	}
	log.Println("<=", version)
	self.Close()
}

func (self *Conn) ReadLine() (line string, err error) {
	return bufio.NewReader(self.sock).ReadString('\n')
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
