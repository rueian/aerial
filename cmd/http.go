package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

var reply string
var delegate string

var httpCmd = &cobra.Command{
	Use: "http",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("http server started at port:", port)
		_ = http.ListenAndServe(":"+strconv.FormatInt(port, 10), http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			var res string
			if reply != "" {
				res = fmt.Sprintf("current time: %s\n%s\n\n", time.Now(), reply)
			}
			if delegate != "" {
				delegation, _ := http.NewRequest(req.Method, "http://"+delegate+req.RequestURI, req.Body)
				delegation.Header = req.Header
				if resp, err := http.DefaultClient.Do(delegation); err == nil {
					rb, _ := ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					res = fmt.Sprintf("delegate to %s:\n%s", delegate, rb)
				} else {
					res = fmt.Sprintf("delegate to %s:\n%s\n\n", delegate, err.Error())
				}
			}
			_, _ = w.Write([]byte(res))
		}))
	},
}

func init() {
	httpCmd.Flags().StringVarP(&reply, "reply", "r", "", "static http reply")
	httpCmd.Flags().StringVarP(&delegate, "delegate", "d", "", "delegate to another host")
	httpCmd.Flags().Int64VarP(&port, "port", "p", 8080, "listen port")
	rootCmd.AddCommand(httpCmd)
}
