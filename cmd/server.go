package cmd

import (
	"encoding/binary"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
)

var port int64

var serverCmd = &cobra.Command{
	Use: "server",
	Run: func(cmd *cobra.Command, args []string) {
		ln, err := net.Listen("tcp", ":"+strconv.FormatInt(port, 10))
		if err != nil {
			log.Fatal(err)
		}
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Println(err)
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()

				req := make([]byte, 5) // 1 byte type + 4 byte length
				if _, err := io.ReadFull(conn, req); err != nil {
					log.Println(err)
					return
				}

				var so net.Listener
				switch req[0] {
				case 't':
					so, err = net.Listen("tcp", ":0")
				case 'd':
					so, err = net.Listen("udp", ":0")
				default:
					return
				}

				if err != nil {
					log.Println(err)
					return
				}

				defer so.Close()

				var p uint32
				switch req[0] {
				case 't':
					p = uint32(so.Addr().(*net.TCPAddr).Port)
				case 'd':
					p = uint32(so.Addr().(*net.UDPAddr).Port)
				}
				head := make([]byte, 9)
				head[0] = 'p'
				binary.BigEndian.PutUint32(head[1:5], 4)
				binary.BigEndian.PutUint32(head[5:9], p)
				if _, err := conn.Write(head); err != nil {
					log.Println(err)
					return
				}

				var mu = sync.Mutex{}
				var sos = map[uint32]net.Conn{}

				// read so, write to conn
				go func() {
					for id := uint32(0); ; id++ {
						cn, err := so.Accept()
						if err != nil {
							log.Println(err)
							return
						}

						mu.Lock()
						sos[id] = cn
						mu.Unlock()

						go func(cn net.Conn, id uint32) {
							defer func() {
								cn.Close()
								mu.Lock()
								delete(sos, id)
								mu.Unlock()
							}()
							buf := make([]byte, 1024)
							for {
								n, err := cn.Read(buf)
								if n > 0 {
									head := make([]byte, 9)
									head[0] = 'm'
									binary.BigEndian.PutUint32(head[1:5], uint32(n)+4)
									binary.BigEndian.PutUint32(head[5:9], id)
									conn.Write(head)
									conn.Write(buf[:n])
								}
								if err == io.EOF {
									return
								}
								if err != nil {
									log.Println(err)
									return
								}
							}
						}(cn, id)
					}
				}()

				for {
					if _, err := io.ReadFull(conn, req); err != nil {
						log.Println(err)
						return
					}
					buf := make([]byte, int(binary.BigEndian.Uint32(req[1:5])))
					if _, err := io.ReadFull(conn, buf); err != nil {
						log.Println(err)
						return
					}
					id := binary.BigEndian.Uint32(buf[:4])
					mu.Lock()
					so, ok := sos[id]
					mu.Unlock()
					if ok {
						so.Write(buf[4:])
					}
				}
			}(conn)
		}
	},
}

func init() {
	serverCmd.Flags().Int64VarP(&port, "port", "p", 8080, "server listen port")
	rootCmd.AddCommand(serverCmd)
}
