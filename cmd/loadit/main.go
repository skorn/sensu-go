package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"strconv"

	"github.com/sensu/sensu-go/agent"
	"github.com/sensu/sensu-go/types"
	"github.com/sirupsen/logrus"

	"net/http"
	_ "net/http/pprof"
)

var (
	flagCount             = flag.Int("count", LookupEnvOrInt("LOADIT_COUNT", 1000), "number of concurrent simulated agents")
	flagBackends          = flag.String("backends", LookupEnvOrString("LOADIT_BACKENDS", "ws://localhost:8081"), "comma separated list of backend URLs")
	flagNamespace         = flag.String("namespace", LookupEnvOrString("LOADIT_NAMESPACE", agent.DefaultNamespace), "namespace to use for agents")
	flagSubscriptions     = flag.String("subscriptions", LookupEnvOrString("LOADIT_SUBSCRIPTIONS", "default"), "comma separated list of subscriptions")
	flagKeepaliveInterval = flag.Int("keepalive-interval", LookupEnvOrInt("LOADIT_KEEPALIVE_INTERVAL", agent.DefaultKeepaliveInterval), "Keepalive interval")
	flagKeepaliveTimeout  = flag.Int("keepalive-timeout", LookupEnvOrInt("LOADIT_KEEPALIVE_TIMEOUT", types.DefaultKeepaliveTimeout), "Keepalive timeout")
	flagProfilingPort     = flag.Int("pprof-port", LookupEnvOrInt("LOADIT_PPROF_PORT", 6060), "pprof port to bind to")
	flagPromBinding       = flag.String("prom", LookupEnvOrString("LOADIT_PROM", ":8080"), "binding for prometheus server")
	flagUser              = flag.String("user", LookupEnvOrString("LOADIT_USER", agent.DefaultUser), "user to authenticate with server")
	flagPassword          = flag.String("password", LookupEnvOrString("LOADIT_PASSWORD", agent.DefaultPassword), "password to authenticate with server")
	flagBaseEntityName    = flag.String("base-entity-name", LookupEnvOrString("LOADIT_ENTITY_NAME", "test-host"), "base entity name to prepend with count number.")
  flagEntityOffset      = flag.Int("entity-offset", LookupEnvOrInt("LOADIT_ENTITY_OFFSET", 0), "password to authenticate with server")
)

func LookupEnvOrString(key string, defaultVal string) string {
       if val, ok := os.LookupEnv(key); ok {
               return val
       }
       return defaultVal
}

func LookupEnvOrInt(key string, defaultVal int) int {
       if val, ok := os.LookupEnv(key); ok {
               v, err := strconv.Atoi(val)
               if err != nil {
                       log.Fatalf("LookupEnvOrInt[%s]: %v", key, err)
               }
               return v
       }
       return defaultVal
}

func main() {
	flag.Parse()

	go func() {
		log.Println(http.ListenAndServe(fmt.Sprintf("localhost:%d", *flagProfilingPort), nil))
	}()

	logrus.SetLevel(logrus.WarnLevel)

	subscriptions := strings.Split(*flagSubscriptions, ",")
	backends := strings.Split(*flagBackends, ",")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	for i := 0; i < *flagCount; i++ {
		name := fmt.Sprintf("%s-%d", *flagBaseEntityName, i+1+*flagEntityOffset)

		cfg := agent.NewConfig()
		cfg.API.Host = agent.DefaultAPIHost
		cfg.API.Port = agent.DefaultAPIPort
		cfg.CacheDir = os.DevNull
		cfg.DisableAssets = true
		cfg.Deregister = true
		cfg.DeregistrationHandler = ""
		cfg.DisableAPI = true
		cfg.DisableSockets = true
		cfg.StatsdServer = &agent.StatsdServerConfig{
			Disable:       true,
			FlushInterval: 10,
		}
		cfg.KeepaliveInterval = uint32(*flagKeepaliveInterval)
		cfg.KeepaliveWarningTimeout = uint32(*flagKeepaliveTimeout)
		cfg.Namespace = *flagNamespace
		cfg.Password = *flagPassword
		cfg.Socket.Host = agent.DefaultAPIHost
		cfg.Socket.Port = agent.DefaultAPIPort
		cfg.User = *flagUser
		cfg.Subscriptions = subscriptions
		cfg.AgentName = name
		cfg.BackendURLs = backends
		cfg.MockSystemInfo = true
		cfg.BackendHeartbeatInterval = 30
		cfg.BackendHeartbeatTimeout = 300
		cfg.PrometheusBinding = *flagPromBinding

		agent, err := agent.NewAgent(cfg)
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			if err := agent.Run(ctx); err != nil {
				log.Fatal(err)
			}
		}()
	}

	elapsed := time.Since(start)
	fmt.Printf("all agents have been connected in %s\n", elapsed)
	fmt.Printf("%s-%d ... %s-%d\n", *flagBaseEntityName, 1+*flagEntityOffset, *flagBaseEntityName, *flagCount+*flagEntityOffset)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
}
