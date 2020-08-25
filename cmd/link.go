package cmd

import (
	"encoding/binary"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net"
	"sync"
)

var addr string
var bind string
var mode string

var linkCmd = &cobra.Command{
	Use: "link",
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Fatal(err)
		}

		req := make([]byte, 5)
		if mode == "udp" {
			req[0] = 'u'
		} else {
			req[0] = 't'
		}
		binary.BigEndian.PutUint32(req[1:], 0)
		if _, err := conn.Write(req); err != nil {
			log.Fatal(err)
		}

		rep := make([]byte, 9)
		if _, err := io.ReadFull(conn, rep); err != nil {
			log.Fatal(err)
		}
		log.Println("server started at ", binary.BigEndian.Uint32(rep[5:]))

		var mu sync.Mutex
		sos := make(map[uint32]net.Conn)

		for {
			if _, err := io.ReadFull(conn, rep); err != nil {
				log.Fatal(err)
			}
			id := binary.BigEndian.Uint32(rep[5:9])
			sz := binary.BigEndian.Uint32(rep[1:5])
			buf := make([]byte, sz-4)
			if _, err := io.ReadFull(conn, buf); err != nil {
				log.Fatal(err)
			}
			mu.Lock()
			so, ok := sos[id]
			mu.Unlock()
			if !ok {
				so, err = net.Dial(mode, bind)
				if err != nil {
					log.Println(err)
					continue
				}
				mu.Lock()
				sos[id] = so
				mu.Unlock()
				go func() {
					defer func() {
						so.Close()
						mu.Lock()
						delete(sos, id)
						mu.Unlock()
					}()

					buf := make([]byte, 1024)
					head := make([]byte, 9)
					head[0] = 'r'
					for {
						n, err := so.Read(buf)
						if n > 0 {
							binary.BigEndian.PutUint32(head[1:5], uint32(n+4))
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
				}()
			}
			if _, err := so.Write(buf); err != nil {
				log.Println(err)
			}
		}
	},
}

func init() {
	linkCmd.Flags().StringVarP(&addr, "addr", "a", "localhost:8080", "aerial server addr")
	linkCmd.Flags().StringVarP(&bind, "bind", "b", "localhost:9999", "target server addr")
	linkCmd.Flags().StringVarP(&mode, "mode", "m", "tcp", "target server proto type")
	rootCmd.AddCommand(linkCmd)
}
