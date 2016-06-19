package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/complainer/mesos"
	"github.com/cloudflare/complainer/monitor"
	"github.com/cloudflare/complainer/reporter"
	"github.com/cloudflare/complainer/uploader"
)

func main() {
	name := flag.String("name", envOrDefault("COMPLAINER_NAME", monitor.DefaultName), "complainer name to use (default is implicit)")
	u := flag.String("uploader", envOrDefault("COMPLAINER_UPLOADER", ""), "uploader to use (example: s3aws,s3goamz,noop)")
	r := flag.String("reporters", envOrDefault("COMPLAINER_REPORTERS", ""), "reporters to use (example: sentry,hipchat,slack,file)")
	masters := flag.String("masters", envOrDefault("COMPLAINER_MASTERS", ""), "list of master urls: http://host:port,http://host:port")

	uploader.RegisterFlags()
	reporter.RegisterFlags()

	flag.Parse()

	if *u == "" || *r == "" || *masters == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	um, err := uploader.MakerByName(*u)
	if err != nil {
		log.Fatalf("Cannot create uploader by name %q: %s", *u, err)
	}

	up, err := um.Make()
	if err != nil {
		flag.PrintDefaults()
		log.Fatalf("Cannot create uploader by name %q: %s", *u, err)
	}

	reporters, err := makeReporters(*r)
	if err != nil {
		log.Fatalf("Cannot create requested reporters: %s", err)
	}

	cluster := mesos.NewCluster(strings.Split(*masters, ","))

	m := monitor.NewMonitor(*name, cluster, up, reporters)

	for {
		err := m.Run()
		if err != nil {
			log.Fatal(err)
		}

		time.Sleep(time.Second * 5)
	}
}

// envOrDefault will return the value of env var key if set.
// If not value is returned
func envOrDefault(key, value string) string {
	if v := os.Getenv(key); len(v) > 0 {
		return v
	}

	return value
}

func makeReporters(requested string) (map[string]reporter.Reporter, error) {
	reporters := map[string]reporter.Reporter{}

	for _, n := range strings.Split(requested, ",") {
		maker, err := reporter.MakerByName(n)
		if err != nil {
			return nil, fmt.Errorf("cannot create reporter by name %q: %s", n, err)
		}

		r, err := maker.Make()
		if err != nil {
			return nil, fmt.Errorf("cannot create reporter by name %q: %s", n, err)
		}

		reporters[n] = r
	}

	return reporters, nil
}
