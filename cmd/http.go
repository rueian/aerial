package cmd

import (
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
)

var reply string

var httpCmd = &cobra.Command{
	Use: "http",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("http server started at port:", port)
		http.ListenAndServe(":"+strconv.FormatInt(port, 10), http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log.Println("handling request from", req.RemoteAddr)
			if reply != "" {
				w.Write([]byte(reply + "\n"))
				return
			}
			res, err := httputil.DumpRequest(req, true)
			if err != nil {
				panic(err)
			}
			w.Write(res)
		}))
	},
}

func init() {
	httpCmd.Flags().StringVarP(&reply, "reply", "r", "", "static http reply")
	httpCmd.Flags().Int64VarP(&port, "port", "p", 8080, "listen port")
	rootCmd.AddCommand(httpCmd)
}
