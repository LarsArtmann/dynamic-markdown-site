package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	cockroachdberrors "github.com/cockroachdb/errors"
)

// Sentinel errors for the healthcheck command.
var (
	errHealthcheckFailed = errors.New("healthcheck failed")
)

// healthcheckSubcommand is the literal subcommand token dispatched from
// main(). It is also a valid argument that runHealthcheck strips before
// parsing its own flags.
const healthcheckSubcommand = "healthcheck"

// runHealthcheck probes the server's /health endpoint and exits 0 on a
// 200 response. The address defaults to localhost:8080; pass --addr to point
// at another host:port.
//
// This is used by container orchestrators (Docker, Kubernetes) that don't
// have access to a shell tool like curl. Distroless images in particular
// have no `wget` or `curl` to back a HEALTHCHECK, so the binary itself
// performs the probe.
func runHealthcheck() error {
	// When invoked from the dispatcher in main() the first argument is
	// the literal "healthcheck" subcommand. Skip it so flag.Parse can
	// consume --addr/--timeout.
	args := os.Args[1:]
	if len(args) > 0 && args[0] == healthcheckSubcommand {
		args = args[1:]
	}

	fs := flag.NewFlagSet(healthcheckSubcommand, flag.ContinueOnError)
	addr := fs.String("addr", "localhost:8080", "host:port to probe")
	timeoutSec := fs.Int("timeout", 5, "timeout in seconds")
	if err := fs.Parse(args); err != nil {
		return cockroachdberrors.Wrap(err, "parse flags")
	}

	url := "http://" + *addr + "/health"

	client := &http.Client{ //nolint:exhaustruct // use defaults
		Timeout: time.Duration(*timeoutSec) * time.Second,
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return cockroachdberrors.Wrapf(errHealthcheckFailed, "build request to %s", url)
	}

	resp, err := client.Do(req)
	if err != nil {
		return cockroachdberrors.Wrapf(errHealthcheckFailed, "GET %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s returned status %d", errHealthcheckFailed, url, resp.StatusCode)
	}

	return nil
}
