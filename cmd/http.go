package cmd

import (
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httputil"
)

var httpCmd = &cobra.Command{
	Use: "http",
	Run: func(cmd *cobra.Command, args []string) {
		http.ListenAndServe(bind, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			res, err := httputil.DumpRequest(req, true)
			if err != nil {
				panic(err)
			}
			w.Write(res)
		}))
	},
}

func init() {
	httpCmd.Flags().StringVarP(&bind, "bind", "b", "0.0.0.0:9999", "target server addr")
	rootCmd.AddCommand(httpCmd)
}
