//go:build windows || (linux && !android)

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/xtls/libxray/dns"
	"github.com/xtls/libxray/xray"
	"github.com/xtls/xray-core/proxy/netbird"
)

func run(options runOptions) error {
	config, err := os.ReadFile(options.configPath)
	if err != nil {
		return err
	}
	if options.interfaceName != "" {
		netbird.SetOutboundInterface(options.interfaceName)
		defer netbird.SetOutboundInterface("")
	}
	if options.setDNS {
		if err := dns.SetDNS(options.dns, options.interfaceName); err != nil {
			return err
		}
		defer dns.ResetDNS()
	}

	if err := xray.RunXray(string(config)); err != nil {
		return err
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	<-signals
	return xray.StopXray()
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "serve-invoke" {
		if err := serveInvoke(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	os.Exit(execute(os.Args[1:], run, os.Stdout, os.Stderr))
}
