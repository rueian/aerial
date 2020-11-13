package cmd

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/rueian/aerial/pkg/hook"
	"github.com/rueian/aerial/pkg/tunnel"
	"github.com/spf13/cobra"
	"log"
	"net"
)

var mirrorCmd = &cobra.Command{
	Use: "mirror",
	Run: func(cmd *cobra.Command, args []string) {
		init := hook.Init{Svc: svc, Params: map[string]string{hook.L7ProtoParamKey: hook.L7ProtoTeePasser}}
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

		for {
			if _, err := msg.ReadFrom(conn); err != nil {
				log.Fatal(err)
			}
			if msg.Type != 'm' {
				continue
			}
			reply := msg.Body[0] == '1'
			ended := msg.Body[1] == '1'
			size := binary.BigEndian.Uint64(msg.Body[2:10])
			str := msg.Body[10 : size+10]

			fmt.Printf("%d %t %t %d %d\n%s\n", msg.Conn, ended, reply, size, len(str), str)
		}
	},
}

func init() {
	mirrorCmd.Flags().StringVarP(&addr, "addr", "a", "localhost:8080", "aerial server addr")
	mirrorCmd.Flags().StringVarP(&svc, "svc", "s", "svc:9999", "cluster target server addr")
	rootCmd.AddCommand(mirrorCmd)
}
