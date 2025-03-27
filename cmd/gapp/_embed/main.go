package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	_ "embed"

	"github.com/rubiojr/gapp/pkg/glance"
	webview "github.com/rubiojr/gapp/vendored/webview"
)

//go:embed glance.yml
var config []byte

func main() {
	host := "127.0.0.1"
	port, err := randomPort()
	if err != nil {
		panic(err)
	}

	opts := []glance.Option{
		glance.WithServerPort(port),
		glance.WithLogger(log.New(os.Stdout, "", log.LstdFlags)),
		glance.WithHost(host),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := glance.ServeApp(ctx, config, opts...)
	if err != nil {
		panic(err)
	}

	w := webview.New(false)
	defer w.Destroy()

	w.SetTitle("Glance Dashboard")
	w.SetSize(1024, 600, webview.HintNone)
	w.Navigate(fmt.Sprintf("http://%s:%d", host, port))
	time.Sleep(time.Second)

	w.Run()
}

func randomPort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}
