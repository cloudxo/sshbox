package main

import (
	"fmt"
	"os"

	shlex "github.com/google/shlex"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	bind    string
	debug   bool
	version bool

	githubAuth bool
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <keys> <command>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVarP(&bind, "bind", "b", ":2222", "interface and port to bind to")
	flag.BoolVarP(&version, "version", "v", false, "display version information")
	flag.BoolVarP(&debug, "debug", "d", false, "enable debug logging")

	flag.BoolVarP(&githubAuth, "github-auth", "g", false, "use github to authorize keys")
}

func main() {
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if version {
		fmt.Printf("bitcaskd version %s", FullVersion())
		os.Exit(0)
	}

	if len(flag.Args()) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	keys := flag.Arg(0)
	cmd := flag.Arg(1)

	args, err := shlex.Split(cmd)
	if err != nil {
		log.WithError(err).Error("error parsing command")
		os.Exit(2)
	}

	cmd = args[0]
	args = args[1:]

	server, err := newServer(
		bind, keys, cmd, args,
		WithGithubAuth(githubAuth),
	)
	if err != nil {
		log.WithError(err).Error("error creating server")
		os.Exit(2)
	}

	log.Infof("sshbox %s listening on %s\n", FullVersion(), bind)
	if err = server.Run(); err != nil {
		log.Fatal(err)
	}
}
