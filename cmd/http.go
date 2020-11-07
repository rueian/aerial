package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
)

var reply string
var delegate string

var httpCmd = &cobra.Command{
	Use: "http",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("http server started at port:", port)
		_ = http.ListenAndServe(":"+strconv.FormatInt(port, 10), http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log.Println("handling request from", req.RemoteAddr)
			var res []byte
			if reply != "" {
				res = []byte(reply + "\n\n")
			} else if delegate != "" {
				_, _ = w.Write([]byte(fmt.Sprintf("delegate to %s:\n", delegate)))
				nreq, _ := http.NewRequest(req.Method, "http://"+delegate+req.RequestURI, req.Body)
				defer req.Body.Close()
				nreq.Header = req.Header
				resp, err := http.DefaultClient.Do(nreq)
				if err != nil {
					res = []byte(err.Error() + "\n\n")
				} else {
					res, _ = ioutil.ReadAll(resp.Body)
					resp.Body.Close()
				}
			} else {
				res, _ = httputil.DumpRequest(req, true)
			}
			_, _ = w.Write(res)
		}))
	},
}

func init() {
	httpCmd.Flags().StringVarP(&reply, "reply", "r", "", "static http reply")
	httpCmd.Flags().StringVarP(&delegate, "delegate", "d", "", "delegate to another host")
	httpCmd.Flags().Int64VarP(&port, "port", "p", 8080, "listen port")
	rootCmd.AddCommand(httpCmd)
}
