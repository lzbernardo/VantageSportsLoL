package main

import (
	"net/http"
	"os"

	"golang.org/x/net/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/VantageSports/common/certs"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolqueue"
	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/payment"
	"github.com/VantageSports/riot/service"
	"github.com/VantageSports/users"
)

var (
	port          = env.Or("PORT", ":80")
	fake          = os.Getenv("FAKE") != ""
	addrLolUsers  = os.Getenv("ADDR_LOLUSERS")
	addrRiotProxy = os.Getenv("ADDR_RIOT_PROXY")
	addrPayment   = os.Getenv("ADDR_PAYMENT")
	addrUsersV2   = os.Getenv("ADDR_USERS_V2")
	tlsCertPath   = os.Getenv("TLS_CERT")
	tlsKeyPath    = os.Getenv("TLS_KEY")

	// temporary option while we transition to self-signed certificates
	insecureGRPC = os.Getenv("INSECURE_GRPC") != ""
)

func init() {
	grpclog.SetLogger(log.NewGRPCAdapter(log.Quiet))
}

func main() {
	vutil := &lolqueue.VantageUtil{Fake: fake}
	if !fake {
		vutil = &lolqueue.VantageUtil{
			Users:    mustUsersClient(),
			LolUsers: mustLolUsersClient(),
			Riot:     mustRiotProxyClient(),
			Payment:  mustPaymentClient(),
		}
	}
	s := lolqueue.NewServer(vutil)

	http.Handle("/connect", websocket.Handler(s.OnConnect))
	http.HandleFunc("/debug", s.Debug)

	log.Info("listening on " + port)
	if tlsCertPath != "" && tlsKeyPath != "" {
		log.Fatal(http.ListenAndServeTLS(port, tlsCertPath, tlsKeyPath, nil))
	} else {
		log.Fatal(http.ListenAndServe(port, nil))
	}
}

func mustUsersClient() users.AuthCheckClient {
	c, err := certs.ClientTLS(tlsCertPath, certs.Insecure(insecureGRPC))
	if err != nil {
		log.Fatal(err)
	}
	conn, err := grpc.Dial(addrUsersV2, grpc.WithTransportCredentials(credentials.NewTLS(c)))
	if err != nil {
		log.Fatal(err)
	}
	return users.NewAuthCheckClient(conn)
}

func mustPaymentClient() payment.PaymentClient {
	c, err := certs.ClientTLS(tlsCertPath, certs.Insecure(insecureGRPC))
	if err != nil {
		log.Fatal(err)
	}
	conn, err := grpc.Dial(addrPayment, grpc.WithTransportCredentials(credentials.NewTLS(c)))
	if err != nil {
		log.Fatal(err)
	}
	return payment.NewPaymentClient(conn)
}

func mustLolUsersClient() lolusers.LolUsersClient {
	c, err := certs.ClientTLS(tlsCertPath, certs.Insecure(insecureGRPC))
	if err != nil {
		log.Fatal(err)
	}
	conn, err := grpc.Dial(addrLolUsers, grpc.WithTransportCredentials(credentials.NewTLS(c)))
	if err != nil {
		log.Fatal(err)
	}
	return lolusers.NewLolUsersClient(conn)
}

func mustRiotProxyClient() service.RiotClient {
	c, err := certs.ClientTLS(tlsCertPath, certs.Insecure(insecureGRPC))
	if err != nil {
		log.Fatal(err)
	}
	conn, err := grpc.Dial(addrRiotProxy, grpc.WithTransportCredentials(credentials.NewTLS(c)))
	if err != nil {
		log.Fatal(err)
	}
	return service.NewRiotClient(conn)
}
