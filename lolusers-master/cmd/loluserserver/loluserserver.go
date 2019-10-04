package main

import (
	"net"
	"net/http"
	"os"

	"github.com/VantageSports/common/certs"
	vscreds "github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue"
	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/lolusers/server"
	"github.com/VantageSports/riot/service"
	"github.com/VantageSports/tasks"
	"github.com/VantageSports/users"
	"github.com/VantageSports/users/client"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"github.com/hashicorp/golang-lru"
	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
)

const defaultPort = ":50001"

func init() {
	grpclog.SetLogger(log.NewGRPCAdapter(log.Quiet))
}

var (
	port             = env.Or("PORT", defaultPort)
	addrUsersV2      = os.Getenv("ADDR_USERS_V2")
	addrRiotProxy    = os.Getenv("ADDR_RIOT_PROXY")
	addrTasks        = env.Must("ADDR_TASKS_EMAIL")
	googleProjectID  = os.Getenv("GOOG_PROJECT_ID")
	internalKey      = env.SmartString("SIGN_KEY_INTERNAL")
	pubsubOutput     = env.Must("PUBSUB_OUTPUT_TOPIC")
	lolActiveGroupID = env.Must("LOL_ACTIVE_GROUP_ID")
	tlsCertPath      = os.Getenv("TLS_CERT")
	tlsKeyPath       = os.Getenv("TLS_KEY")

	// temporary option while we transition to self-signed certificates
	insecureGRPC = os.Getenv("INSECURE_GRPC") != ""
)

func main() {
	// debugging via http /debug/{events,requests}
	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) { return true, true }
	go http.ListenAndServe(":9009", nil)

	lis, err := net.Listen("tcp", port)
	exitIf(err)

	if tlsCertPath == "" {
		tlsCertPath, tlsKeyPath = certs.MustWriteDevCerts()
	}
	creds, err := credentials.NewServerTLSFromFile(tlsCertPath, tlsKeyPath)
	exitIf(err)
	s := grpc.NewServer(grpc.Creds(creds))

	registerLolUserService(s)

	log.Notice("starting on: " + port)
	s.Serve(lis)
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func registerLolUserService(s *grpc.Server) {
	pubsubClient := mustPubsubClient()

	svc := &server.LolUserService{
		RiotProxy:           mustRiotProxyClient(),
		GcdClient:           mustDatastoreClient(),
		AuthChecker:         mustAuthClient(),
		CrawlSummonersCache: mustCrawlCache(100),
		PubTopic:            mustPubTopic(pubsubClient),
		Users:               mustUsersClient(),
		Emailer:             mustEmailClient(),
		InternalKey:         internalKey,
		LolActiveGroupID:    lolActiveGroupID,
	}
	logs := &LolUsersLogger{svc}
	lolusers.RegisterLolUsersServer(s, logs)
}

func mustPubTopic(client *pubsub.Client) *pubsub.Topic {
	topicHandle, err := queue.EnsureTopic(client, pubsubOutput)
	exitIf(err)
	return topicHandle
}

func mustPubsubClient() *pubsub.Client {
	creds := vscreds.MustEnvCreds(googleProjectID, pubsub.ScopePubSub)
	client, err := queue.InitClient(creds)
	exitIf(err)
	return client
}

func mustAuthClient() users.AuthCheckClient {
	client, err := client.DialAuthCheck(addrUsersV2, tlsCertPath, internalKey, 100, insecureGRPC)
	exitIf(err)
	return client
}

func mustUsersClient() users.UsersClient {
	c, err := certs.ClientTLS(tlsCertPath, certs.Insecure(insecureGRPC))
	if err != nil {
		log.Fatal(err)
	}
	conn, err := grpc.Dial(addrUsersV2, grpc.WithTransportCredentials(credentials.NewTLS(c)))
	if err != nil {
		log.Fatal(err)
	}
	return users.NewUsersClient(conn)
}

func mustCrawlCache(cacheSize int) *lru.Cache {
	cache, err := lru.New(cacheSize)
	exitIf(err)
	return cache
}

func mustDatastoreClient() *datastore.Client {
	ctx := context.Background()
	creds := vscreds.MustEnvCreds(googleProjectID, datastore.ScopeDatastore)

	client, err := datastore.NewClient(ctx, googleProjectID, option.WithTokenSource(creds.TokenSource(ctx)))
	exitIf(err)
	return client
}

func mustRiotProxyClient() service.RiotClient {
	c, err := certs.ClientTLS(tlsCertPath, certs.Insecure(insecureGRPC))
	exitIf(err)
	conn, err := grpc.Dial(addrRiotProxy, grpc.WithTransportCredentials(credentials.NewTLS(c)))
	exitIf(err)
	return service.NewRiotClient(conn)
}

func mustEmailClient() tasks.EmailClient {
	c, err := certs.ClientTLS(tlsCertPath, certs.Insecure(insecureGRPC))
	if err != nil {
		log.Fatal(err)
	}
	conn, err := grpc.Dial(addrTasks, grpc.WithTransportCredentials(credentials.NewTLS(c)))
	if err != nil {
		log.Fatal(err)
	}
	return tasks.NewEmailClient(conn)
}
