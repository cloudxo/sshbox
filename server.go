package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/gliderlabs/ssh"
	"github.com/kr/pty"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	githubKeysURL  = "https://github.com/%s.keys"
	defaultEnvPath = "/sbin:/usr/sbin:/bin:/usr/bin"
	defaultEnvHome = "/var/empty"
)

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

func fetchGithubKeys(ctx context.Context, username string) ([]ssh.PublicKey, error) {
	keyURL := fmt.Sprintf(githubKeysURL, username)

	req, err := http.NewRequest("GET", keyURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating request")
	}

	req = req.WithContext(ctx)

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "fetching keys")
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, errors.New("invalid response from github")
	}

	authorizedKeysBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading body")
	}

	keys := []ssh.PublicKey{}
	for len(authorizedKeysBytes) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			return nil, errors.Wrap(err, "parsing key")
		}

		keys = append(keys, pubKey)
		authorizedKeysBytes = rest
	}

	return keys, nil
}

type option func(*server) error

func WithGithubAuth(githubAuth bool) option {
	return func(s *server) error {
		s.githubAuth = githubAuth
		return nil
	}
}

type server struct {
	bind           string
	cmd            string
	args           []string
	authorizedkeys []ssh.PublicKey

	githubAuth bool
}

func newServer(bind, keys, cmd string, args []string, opts ...option) (*server, error) {
	var authorizedkeys []ssh.PublicKey

	if strings.HasPrefix(keys, "file://") || fileExists(keys) {
		keys = strings.TrimPrefix(keys, "file://")
		data, err := ioutil.ReadFile(keys)
		if err != nil {
			return nil, fmt.Errorf("error reading keys file: %w", err)
		}

		for _, key := range bytes.Split(data, []byte("\n")) {
			publickey, _, _, _, _ := ssh.ParseAuthorizedKey(key)
			authorizedkeys = append(authorizedkeys, publickey)
		}
	}

	s := &server{
		bind:           bind,
		cmd:            cmd,
		args:           args,
		authorizedkeys: authorizedkeys,
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, fmt.Errorf("error applying option: %w", err)
		}
	}

	return s, nil
}

func (s *server) sessionHandler(sess ssh.Session) {
	stime := time.Now()
	log.Infof(
		"New session @%s %s (%d)",
		sess.User(), sess.RemoteAddr(), stime.Unix(),
	)
	defer func() {
		etime := time.Now()
		dtime := etime.Sub(stime)
		log.Infof(
			"Session ended @%s %s (%d) [%s]",
			sess.User(), sess.RemoteAddr(), etime.Unix(), dtime,
		)
	}()

	cmd := exec.Command(s.cmd, s.args...)
	log.Debugf("Executing command %s with arguments %s", s.cmd, s.args)

	cmd.Env = append(cmd.Env, []string{
		fmt.Sprintf("PATH=%s", defaultEnvPath),
		fmt.Sprintf("HOME=%s", defaultEnvHome),
	}...)

	ptyReq, winCh, isPty := sess.Pty()
	if isPty {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
		f, err := pty.Start(cmd)
		if err != nil {
			log.WithError(err).Error("error executing command")
			sess.Exit(1)
			return
		}
		go func() {
			for win := range winCh {
				setWinsize(f, win.Width, win.Height)
			}
		}()
		go func() {
			io.Copy(f, sess) // stdin
		}()
		io.Copy(sess, f) // stdout
		cmd.Wait()
	} else {
		io.WriteString(sess, "No PTY requested.\n")
		sess.Exit(1)
	}
}

func (s *server) Shutdown() (err error) {
	return
}

func (s *server) Run() (err error) {
	sshServer := &ssh.Server{
		Addr:    s.bind,
		Handler: s.sessionHandler,
	}

	sshServer.SetOption(
		ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			user := ctx.User()

			if s.githubAuth {
				keys, err := fetchGithubKeys(ctx, user)
				if err != nil {
					log.WithError(err).Warnf("error fetching Github for %s", user)
				} else {
					for _, key := range keys {
						// TODO: Prevent dpulicate keys
						s.authorizedkeys = append(s.authorizedkeys, key)
					}
				}
			}

			for _, publickey := range s.authorizedkeys {
				if ssh.KeysEqual(key, publickey) {
					log.Infof("User %s authorized", user)
					return true
				}
			}
			log.Warnf("User %s denied", user)
			return false
		}),
	)

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		s := <-signals
		log.Infof("Shutdown server on signal %s", s)
		sshServer.Close()
	}()

	if err := sshServer.ListenAndServe(); err != nil {
		return s.Shutdown()
	}
	return
}
