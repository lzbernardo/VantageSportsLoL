package main

import (
	"net"
	"net/http"
	"os"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	vsbigquery "github.com/VantageSports/common/bigquery"
	"github.com/VantageSports/common/certs"
	vscreds "github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats"
	"github.com/VantageSports/lolstats/server"
	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/users"
	"github.com/VantageSports/users/client"
)

var (
	addr            = env.Or("ADDR", ":50000")
	addrUsersV2     = os.Getenv("ADDR_USERS_V2")
	googleProjectID = env.Must("GOOG_PROJECT_ID")
	internalKey     = env.SmartString("SIGN_KEY_INTERNAL")
	tlsCert         = os.Getenv("TLS_CERT")
	tlsKey          = os.Getenv("TLS_KEY")
	lolUsersAddr    = env.Must("LOL_USERS_SERVER_ADDR")
	basicTable      = env.Must("BIG_QUERY_TABLE_BASIC")
	advancedTable   = env.Must("BIG_QUERY_TABLE_ADVANCED")

	// temporary option while we transition to self-signed certificates
	insecureGRPC = os.Getenv("INSECURE_GRPC") != ""
)

func init() {
	grpclog.SetLogger(log.NewGRPCAdapter(log.Quiet))
}

func main() {
	// debugging via http /debug/{events,requests}
	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) { return true, true }
	go http.ListenAndServe(":9000", nil)

	if tlsCert == "" {
		tlsCert, tlsKey = certs.MustWriteDevCerts()
	}
	creds, err := credentials.NewServerTLSFromFile(tlsCert, tlsKey)
	if err != nil {
		log.Fatal(err)
	}

	svc := &server.GoalsServer{
		AuthClient:     mustAuthClient(),
		DsClient:       mustDatastoreClient(),
		BqClient:       mustBqClient(),
		BasicTable:     basicTable,
		AdvancedTable:  advancedTable,
		LolUsersClient: mustLolUsersClient(tlsCert, lolUsersAddr),
		ProjectID:      googleProjectID,
	}

	s := grpc.NewServer(grpc.Creds(creds))
	lolstats.RegisterLolgoalsServer(s, &GoalsLogger{svc})

	log.Notice("starting on: " + addr)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(s.Serve(lis))
}

func mustDatastoreClient() *datastore.Client {
	ctx := context.Background()
	creds := vscreds.MustEnvCreds(googleProjectID, datastore.ScopeDatastore)

	client, err := datastore.NewClient(ctx, googleProjectID, option.WithTokenSource(creds.TokenSource(ctx)))
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func mustBqClient() *vsbigquery.Client {
	creds := vscreds.MustEnvCreds(googleProjectID, bigquery.Scope)
	bqClient, err := vsbigquery.NewClient(creds)
	if err != nil {
		log.Fatal(err)
	}
	return bqClient
}

func mustAuthClient() users.AuthCheckClient {
	log.Debug(addrUsersV2)
	authClient, err := client.DialAuthCheck(addrUsersV2, tlsCert, internalKey, 100, insecureGRPC)
	if err != nil {
		log.Fatal(err)
	}
	return authClient
}

func mustLolUsersClient(serverCertPath, serverAddr string) lolusers.LolUsersClient {
	config, err := certs.ClientTLS(serverCertPath, certs.Insecure(insecureGRPC))
	if err != nil {
		log.Fatal(err)
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(config)))
	if err != nil {
		log.Fatal(err)
	}

	return lolusers.NewLolUsersClient(conn)
}
