package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
	"unsafe"

	"github.com/gliderlabs/ssh"
	"github.com/kr/pty"
	log "github.com/sirupsen/logrus"
)

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

type server struct {
	bind           string
	cmd            string
	args           []string
	authorizedkeys []ssh.PublicKey
}

func newServer(bind, keys, cmd string, args []string) (*server, error) {
	var authorizedkeys []ssh.PublicKey

	data, err := ioutil.ReadFile(keys)
	if err != nil {
		return nil, fmt.Errorf("error reading keys file: %w", err)
	}

	for _, key := range bytes.Split(data, []byte("\n")) {
		publickey, _, _, _, _ := ssh.ParseAuthorizedKey(key)
		authorizedkeys = append(authorizedkeys, publickey)
	}

	return &server{
		bind:           bind,
		cmd:            cmd,
		args:           args,
		authorizedkeys: authorizedkeys,
	}, nil
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

	ptyReq, winCh, isPty := sess.Pty()
	if isPty {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
		f, err := pty.Start(cmd)
		if err != nil {
			panic(err)
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
			for _, publickey := range s.authorizedkeys {
				if ssh.KeysEqual(key, publickey) {
					return true
				}
			}
			return false
		},
		),
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
