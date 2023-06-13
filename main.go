package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

const (
	serverAddr = "localhost:8443"
)

var (
	serverQuic http3.Server
	httpRouter = mux.NewRouter()
)

func tlsConfig() *tls.Config {
	// caCertPool := x509.NewCertPool()
	// Create a CA certificate pool and add cert.pem to it

	// caCert, err := ioutil.ReadFile(Config.Webserver.ExternalUsersCA)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// caCertPool.AppendCertsFromPEM(caCert)

	tlsCFG := &tls.Config{
		// MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		// CipherSuites:             cipherSuitesSlice,
		// ClientCAs:  caCertPool,
		// ClientAuth: tls.VerifyClientCertIfGiven,
	}

	cert, err := tls.LoadX509KeyPair("ss.crt", "ss.key")
	if err != nil {
		log.Fatal(err)
	}
	tlsCFG.Certificates = append(tlsCFG.Certificates, cert)

	return tlsCFG

}

type zeroHandler struct {
	h http.Handler
	n int64
}

func main() {}

func init() {
	setLimit()
	go pprofInit()
	go openManyConnections()
	hServer := &http.Server{
		Handler:   &zeroHandler{h: http.AllowQuerySemicolons(httpRouter), n: int64(7 << 20)}, //data limit size
		Addr:      serverAddr,
		TLSConfig: tlsConfig(),
		// TLSNextProto:      make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		WriteTimeout:      time.Second * 20,
		ReadTimeout:       time.Second * 11, // ReadTimeout is the maximum duration for reading the entire request, including the body. //! R&D needed
		IdleTimeout:       time.Second * 60, //keep-alive timeout
		MaxHeaderBytes:    5 << 10,          //maximum number of bytes the server will read parsing the request header's keys and values, including the request line.
		ReadHeaderTimeout: time.Second * 11, //ReadHeaderTimeout is the amount of time allowed to read request headers.
	}
	go listenQuic(hServer)
	listenHTTP(hServer)
}

func setLimit() {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}

	log.Printf("set cur limit: %d", rLimit.Cur)
}
func listenHTTP(httpServer *http.Server) {
	if err := httpServer.ListenAndServeTLS("", ""); err != nil {
		log.Printf("HTTP Server error: %v", err)
	}
}
func listenQuic(httpServer *http.Server) {
	var SupportedVersions = []quic.VersionNumber{quic.VersionDraft29, quic.Version1, quic.Version2}
	quicConf := &quic.Config{EnableDatagrams: true, Versions: SupportedVersions}

	//----------------------------------quic Debug--------------------------------------------
	// quicConf.Tracer = func(ctx context.Context, p logging.Perspective, connID quic.ConnectionID) logging.ConnectionTracer {
	// 	filename := fmt.Sprintf("/tmp/server_%x.qlog", connID)
	// 	f, err := os.Create(filename)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	log.Printf("Creating qlog file %s.\n", filename)
	// 	return qlog.NewConnectionTracer(NewBufferedWriteCloser(bufio.NewWriter(f), f), p, connID)
	// }
	//-----------------------------------------------------------------------------------------
	serverQuic = http3.Server{
		Handler:         httpServer.Handler,
		Addr:            httpServer.Addr,
		QuicConfig:      quicConf,
		EnableDatagrams: true,
		MaxHeaderBytes:  httpServer.MaxHeaderBytes,
		TLSConfig:       httpServer.TLSConfig,
	}
	// f, err := os.Create("/tmp/webserverLog.txt")
	// defer f.Close()
	// if err != nil {
	// 	panic(err)
	// }
	// log.SetOutput(f)
	if err := serverQuic.ListenAndServe(); err != nil {
		log.Printf("Quic Server error: %v", err)
	}

}
