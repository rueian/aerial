package cmd

import (
	"encoding/json"
	"github.com/rueian/aerial/pkg/buffer"
	"github.com/rueian/aerial/pkg/hook"
	"github.com/rueian/aerial/pkg/tunnel"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

var addr string
var bind string
var svc string
var params []string

var linkCmd = &cobra.Command{
	Use: "link",
	Run: func(cmd *cobra.Command, args []string) {
		init := hook.Init{
			Svc:    svc,
			Params: map[string]string{},
		}
		for _, v := range params {
			vp := strings.SplitN(v, "=", 2)
			init.Params[vp[0]] = vp[1]
		}

		bs, err := json.Marshal(init)
		if err != nil {
			log.Fatal(err)
		}

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Fatal(err)
		}

		msg := tunnel.Message{Type: 't', Body: bs}
		if _, err := msg.WriteTo(conn); err != nil {
			log.Fatal(err)
		}

		if _, err := msg.ReadFrom(conn); err != nil {
			log.Fatal(err)
		}
		if msg.Conn == 0 {
			log.Println("server error: ", string(msg.Body))
			return
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
					msg := tunnel.Message{Type: 'c', Conn: msg.Conn}
					msg.WriteTo(conn)
					continue
				}
				log.Println("redirect", so.(net.Conn).RemoteAddr())
				sos.Store(msg.Conn, so)
				go func(id uint32) {
					defer func(id uint32) {
						msg := tunnel.Message{Type: 'c', Conn: id}
						msg.WriteTo(conn)
						so.(net.Conn).Close()
						sos.Delete(id)
						log.Println("close", so.(net.Conn).RemoteAddr())
					}(id)

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
							return
						}
					}
				}(msg.Conn)
			}
			if _, err := so.(net.Conn).Write(msg.Body); err != nil {
				log.Println(err)
			}
			if msg.Type == 'c' {
				so.(net.Conn).Close()
				sos.Delete(msg.Conn)
			}
		}
	},
}

func init() {
	linkCmd.Flags().StringVarP(&addr, "addr", "a", "localhost:8080", "aerial server addr")
	linkCmd.Flags().StringVarP(&bind, "bind", "b", "localhost:9999", "local target server addr")
	linkCmd.Flags().StringVarP(&svc, "svc", "s", "svc:9999", "cluster target server addr")
	linkCmd.Flags().StringArrayVarP(&params, "param", "p", nil, "--param key=value")
	rootCmd.AddCommand(linkCmd)
}
