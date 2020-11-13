package cmd

import (
	"github.com/rueian/aerial/pkg/buffer"
	"github.com/rueian/aerial/pkg/hook"
	"github.com/rueian/aerial/pkg/tunnel"
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

		log.Println("tunnel server started at port:", port)

		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Println(err)
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()

				msg := tunnel.Message{}
				if _, err := msg.ReadFrom(conn); err != nil {
					log.Println(err)
					return
				}

				so, err := net.Listen("tcp", ":0")
				if err != nil {
					log.Println(err)
					return
				}
				defer so.Close()

				if ctx, err := hook.OnBind(msg, so.Addr()); err != nil {
					log.Println(err)
					msg = tunnel.Message{Type: 'p', Body: []byte(err.Error())}
					_, _ = msg.WriteTo(conn)
					return
				} else {
					defer hook.OnClose(ctx)
				}

				msg = tunnel.Message{Type: 'p', Conn: uint32(so.Addr().(*net.TCPAddr).Port)}
				if _, err := msg.WriteTo(conn); err != nil {
					log.Println(err)
					return
				}

				var sos = sync.Map{}
				go func() {
					for id := uint32(0); ; id++ {
						cn, err := so.Accept()
						if err != nil {
							log.Println(err)
							return
						}
						sos.Store(id, cn)

						go func(cn net.Conn, id uint32) {
							log.Println("redirecting", cn.LocalAddr(), cn.RemoteAddr(), id)
							defer func(cn net.Conn, id uint32) {
								msg := tunnel.Message{Type: 'c', Conn: id}
								msg.WriteTo(conn)
								cn.Close()
								sos.Delete(id)
								log.Println("close", cn.LocalAddr(), cn.RemoteAddr(), id)
							}(cn, id)
							for {
								buf := buffer.PoolK.Get()
								n, err := cn.Read(buf)
								if n > 0 {
									msg := tunnel.Message{Type: 'm', Conn: id, Body: buf[:n]}
									_, err = msg.WriteTo(conn)
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
						}(cn, id)
					}
				}()

				for {
					if _, err := msg.ReadFrom(conn); err != nil {
						if err != io.EOF {
							log.Println(err)
						}
						return
					}
					if conn, ok := sos.Load(msg.Conn); ok {
						conn := conn.(net.Conn)
						conn.Write(msg.Body)
						if msg.Type == 'c' {
							conn.Close()
							sos.Delete(msg.Conn)
						}
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
