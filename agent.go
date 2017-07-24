package main

import (
	"net"
	"log"
	"sync"
	"flag"
)

var saddr string
var daddr string
var buffer_pool sync.Pool

func init() {
	flag.StringVar(&saddr, "saddr", "localhost:8080", "source addr")
	flag.StringVar(&daddr, "daddr", "", "dest addr")
	flag.Parse()
	if daddr=="" {
		log.Fatal("must set dest addr")
	}
	buffer_pool.New = func() interface{}{
		return make([]byte, 2048)
	}
}
func getBuffer() []byte {
	return buffer_pool.Get().([]byte)
}
func putBuffer(b []byte)  {
	buffer_pool.Put(b)
}
func serveTcp(wg *sync.WaitGroup)  {
	l_tcp, err := net.Listen("tcp", saddr)
	if err != nil {
		log.Println("[tcp]:", err)
		return
	}
	defer func() {
		l_tcp.Close()
		wg.Done()
	}()
	log.Printf("listen tcp '%s' ok, transport to '%s'\n", l_tcp.Addr(), daddr)
	handleTcp(l_tcp)
}
func handleTcp(l net.Listener)  {
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept error: %s\n", err)
			continue
		}
		go handleTcpConn(conn)
	}
}
func copy(source, sink net.Conn, buffer []byte, wg *sync.WaitGroup)  {
	var er, ew error
	var n int
	defer func() {
		wg.Done()
	}()
	for {
		n, er = source.Read(buffer)
		if n>0 {
			_, ew= sink.Write(buffer[:n])
		}
		// if any error happens, break
		if er != nil || ew != nil {
			log.Printf("err=(%s,%s)\n", er, ew)
			break
		}
	}
}
func handleTcpConn(sConn net.Conn)  {
	dConn, err := net.Dial("tcp", daddr)
	if err != nil {
		log.Printf("tcp dial for '%s' err:%s\n", sConn.RemoteAddr(), err)
		sConn.Close()
		return
	}
	to_dest := getBuffer()
	to_src := getBuffer()
	defer func() {
		if r := recover(); r!=nil {
			log.Printf("panic(%s) recoverd for [%s] => [%s]\n", r, sConn.RemoteAddr(), dConn.RemoteAddr())
		}
		sConn.Close()
		dConn.Close()

		putBuffer(to_dest)
		putBuffer(to_src)
	}()
	var wg sync.WaitGroup
	wg.Add(2)
	go copy(sConn, dConn, to_dest, &wg)
	go copy(dConn, sConn, to_src, &wg)

	wg.Wait()
}
func serveUdp(wg *sync.WaitGroup)  {
	addr, err := net.ResolveUDPAddr("udp", saddr)
	if err != nil {
		log.Printf("resolve addr(%s) for udp failed\n", err)
		return
	}
	l, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Println("[udp]:", err)
		return
	}
	//log.Printf("listen udp at '%s'\n", addr)
	log.Printf("listen tcp @%s ok, transport to '%s'\n", addr, daddr)
	defer func() {
		l.Close()
		wg.Done()
	}()
	handleUdp(l)
}
func handleUdp(l *net.UDPConn)  {
	// it is a bit different with tcp,
	// do it later
}
func main()  {
	log.Println("WTF")
	var wg sync.WaitGroup
	wg.Add(2)
	go serveTcp(&wg)
	go serveUdp(&wg)
	wg.Wait()
}
