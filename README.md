# boxssh

A very simple SSh server that "boxes" you into a sandbox environment using
[uLinux](https://github.com/prologic/ulinux) `box` utility for creating
lightweight sandboxes.

> Based on [this example](https://github.com/gliderlabs/ssh/tree/master/_examples/ssh-pty)

## Features

`sshbox` supports:

* __Authorized Key__ for authenticating sessions
* __Github__ for authenticating sessions
* __Sandboxing__ by running `box` or `docker` to sandbox sessions

## Install

If you have a Go development environment setup with `$GOPATH/bin/` in your `$PATH`
the following will just work™ 😀

```#!console
go get -u github.com/prologic/sshbox
```

Otherwise you can build from source using `git` (_You still need the Go compiler_):

```#!console
git clone https://github.com/prologic/sshbox.git
cd sshbox
make
```

### Prebuilt Binaries

There are prebuilt binaries I publish regularly to the
[Releases](https://github.com/prologic/sshbox/releases) page you can download
and install. Example:

```#!console
wget https://github.com/prologic/sshbox/releases/download/0.0.2/sshbox_0.0.2_linux_amd64.tar.gz
tar xvf sshbox_0.0.2_linux_amd64.tar.gz
```

## Usage

Run an SSH server on the default port listening port `;2222`, authorising
users with an `authorized_keys` file  and sandbox user sessions using `box`
with an Alpine Linux container:

```#!console
sshbox ~/.ssh/authorized_keys 'box run alpine /bin/sh'
```

Same thing but authorize users via their Github SSH Keys:

```#!console
sshbox -g /dev/null 'box run alpine /bin/sh'
```

__NOTE:__ Only tested on [uLinux](https://github.com/prologic/ulinux).

> This _may_ work on your system, your milage may vary. File an issue or pull
> request if you have any questions, issues or improvements.

## License

`sshbox` is licensed under the terms of the MIT License
