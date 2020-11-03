package hook

import "net"

func OnBind(addr net.Addr) (interface{}, error) {
	return nil, nil
}

func OnClose(interface{}) {

}
