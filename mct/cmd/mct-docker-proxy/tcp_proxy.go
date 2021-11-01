/*
   Copyright 2013-2018 Docker, Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       https://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"sync"
)

// TCPProxy is a proxy for TCP connections. It implements the Proxy interface to
// handle TCP traffic forwarding between the frontend and backend addresses.
type TCPProxy struct {
	listener     *net.TCPListener
	frontendAddr *net.TCPAddr
	backendAddr  *net.TCPAddr
}

// NewTCPProxy creates a new TCPProxy.
func NewTCPProxy(frontendAddr, backendAddr *net.TCPAddr) (*TCPProxy, error) {
	// detect version of hostIP to bind only to correct version
	ipVersion := ipv4
	if frontendAddr.IP.To4() == nil {
		ipVersion = ipv6
	}
	listener, err := net.ListenTCP("tcp"+string(ipVersion), frontendAddr)
	if err != nil {
		return nil, err
	}

	// expose using gvisor-tap-vsock
	u, err := url.Parse(fmt.Sprintf("http://%s:%s/services/forwarder/expose", getAPIEndpoint(), apiEndpointPort))
	if err != nil {
		return nil, err
	}
	// only ipv4
	if frontendAddr.IP.To4() != nil {
		expose := Expose{
			Local:    fmt.Sprintf("%s:%d", "0.0.0.0", listener.Addr().(*net.TCPAddr).Port),
			Remote:   fmt.Sprintf("%s:%d", "192.168.127.2", listener.Addr().(*net.TCPAddr).Port),
			Protocol: "tcp",
		}
		log.Printf("gvisor-tap-vsock: expose %+v\n", expose)
		if err := postRequest(context.Background(), u, expose); err != nil {
			return nil, err
		}
	} else {
		log.Printf("gvisor-tap-vsock: ignoring ipv6: %+v\n", frontendAddr)
	}
	// If the port in frontendAddr was 0 then ListenTCP will have a picked
	// a port to listen on, hence the call to Addr to get that actual port:
	return &TCPProxy{
		listener:     listener,
		frontendAddr: listener.Addr().(*net.TCPAddr),
		backendAddr:  backendAddr,
	}, nil
}

func (proxy *TCPProxy) clientLoop(client *net.TCPConn, quit chan bool) {
	backend, err := net.DialTCP("tcp", nil, proxy.backendAddr)
	if err != nil {
		log.Printf("Can't forward traffic to backend tcp/%v: %s\n", proxy.backendAddr, err)
		client.Close()
		return
	}

	var wg sync.WaitGroup
	var broker = func(to, from *net.TCPConn) {
		io.Copy(to, from)
		from.CloseRead()
		to.CloseWrite()
		wg.Done()
	}

	wg.Add(2)
	go broker(client, backend)
	go broker(backend, client)

	finish := make(chan struct{})
	go func() {
		wg.Wait()
		close(finish)
	}()

	select {
	case <-quit:
	case <-finish:
	}
	client.Close()
	backend.Close()
	<-finish
}

// Run starts forwarding the traffic using TCP.
func (proxy *TCPProxy) Run() {
	quit := make(chan bool)
	defer close(quit)
	unexpose := func() {
		u, err := url.Parse(fmt.Sprintf("http://%s:%s/services/forwarder/unexpose", getAPIEndpoint(), apiEndpointPort))
		if err != nil {
			log.Printf("failed to parse url: %v", err)
			return
		}
		unexpose := Unexpose{Local: fmt.Sprintf("0.0.0.0:%d", proxy.frontendAddr.Port)}
		if err := postRequest(context.Background(), u, unexpose); err != nil {
			log.Print(err)
			return
		}
	}
	defer unexpose()
	for {
		client, err := proxy.listener.Accept()
		if err != nil {
			log.Printf("Stopping proxy on tcp/%v for tcp/%v (%s)", proxy.frontendAddr, proxy.backendAddr, err)
			return
		}
		go proxy.clientLoop(client.(*net.TCPConn), quit)
	}
}

// Close stops forwarding the traffic.
func (proxy *TCPProxy) Close() {
	proxy.listener.Close()
}

// FrontendAddr returns the TCP address on which the proxy is listening.
func (proxy *TCPProxy) FrontendAddr() net.Addr { return proxy.frontendAddr }

// BackendAddr returns the TCP proxied address.
func (proxy *TCPProxy) BackendAddr() net.Addr { return proxy.backendAddr }
