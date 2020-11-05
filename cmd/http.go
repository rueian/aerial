package cmd

import (
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httputil"
	"strconv"
)

var reply string

var httpCmd = &cobra.Command{
	Use: "http",
	Run: func(cmd *cobra.Command, args []string) {
		http.ListenAndServe(":"+strconv.FormatInt(port, 10), http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if reply != "" {
				w.Write([]byte(reply))
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
	linkCmd.Flags().StringVarP(&reply, "reply", "r", "", "static http reply")
	rootCmd.AddCommand(httpCmd)
}
