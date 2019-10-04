package http

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
)

// StartTLSServer starts an https server for the supplied handler on the
// supplied port with the crt and keys found in the file locations supplied.
func StartTLSServer(addr, sslCrtPath, sslKeyPath string, handler http.Handler) error {
	for _, filename := range []string{sslCrtPath, sslKeyPath} {
		if _, err := os.Stat(filename); err != nil {
			return err
		}
	}
	log.Println("Starting HTTPS server on", addr)
	// Instantiate a server (rather than just calling ListenAndServeTLS)
	// so that we set a hardened TLSConfig.
	s := &http.Server{Addr: addr, Handler: handler}
	s.TLSConfig = newTLSConfig()
	return s.ListenAndServeTLS(sslCrtPath, sslKeyPath) // Blocks
}

// StartServer starts an http server for the supplied handler on the supplied
// port. Sort of a dumb wrapper around ListenAndServe, but provided in this
// package for symmetry with StartTLSServer
func StartServer(addr string, handler http.Handler) error {
	// Otherwise, just start a plain HTTP server.
	log.Println("Starting HTTP server on", addr)
	return http.ListenAndServe(addr, handler) // Blocks
}

// Starts an http server on the supplied addr (derived from the "PORT" env var,
// and if either "SSL_CRT_FILE" or "SSL_KEY_FILE" are set, will attempt to
// start a TLS-enabled server.
func StartFromEnv(handler http.Handler) error {
	port := os.Getenv("PORT")
	certFile := os.Getenv("SSL_CRT_FILE")
	keyFile := os.Getenv("SSL_KEY_FILE")
	if certFile != "" || keyFile != "" {
		return StartTLSServer(port, certFile, keyFile, handler)
	}
	return StartServer(port, handler)
}

// newTLSConfig creates a TLS config for a server, removing the vulerabilities
// identified by running https://www.ssllabs.com/ssltest/.
func newTLSConfig() *tls.Config {
	c := &tls.Config{}
	c.MinVersion = tls.VersionTLS10   // Prevent TLS v3
	c.PreferServerCipherSuites = true // Express cipher preference
	c.CipherSuites = []uint16{        // Turn off the RC4 (broken) suites
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_FALLBACK_SCSV,
	}
	return c
}
