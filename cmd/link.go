package cmd

import (
	"github.com/rueian/aerial/pkg/buffer"
	"github.com/rueian/aerial/pkg/tunnel"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net"
	"sync"
)

var addr string
var bind string

var linkCmd = &cobra.Command{
	Use: "link",
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Fatal(err)
		}

		msg := tunnel.Message{Type: 't'}
		if _, err := msg.WriteTo(conn); err != nil {
			log.Fatal(err)
		}

		if _, err := msg.ReadFrom(conn); err != nil {
			log.Fatal(err)
		}
		log.Println("server started at", msg.Conn)

		sos := sync.Map{}
		for {
			if _, err := msg.ReadFrom(conn); err != nil {
				log.Fatal(err)
			}
			so, ok := sos.Load(msg.Conn)
			if !ok {
				so, err = net.Dial("tcp", bind)
				if err != nil {
					log.Println(err)
					continue
				}
				log.Println("redirecting", bind)
				sos.Store(msg.Conn, so)
				go func() {
					defer func() {
						so.(net.Conn).Close()
						sos.Delete(msg.Conn)
					}()

					for {
						buf := buffer.PoolK.Get()
						n, err := so.(net.Conn).Read(buf)
						if n > 0 {
							msg := tunnel.Message{Type: 'r', Conn: msg.Conn, Body: buf[:n]}
							msg.WriteTo(conn)
						}
						buffer.PoolK.Put(buf)
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
			if _, err := so.(net.Conn).Write(msg.Body); err != nil {
				log.Println(err)
			}
		}
	},
}

func init() {
	linkCmd.Flags().StringVarP(&addr, "addr", "a", "localhost:8080", "aerial server addr")
	linkCmd.Flags().StringVarP(&bind, "bind", "b", "localhost:9999", "target server addr")
	rootCmd.AddCommand(linkCmd)
}
