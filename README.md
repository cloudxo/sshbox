# boxssh

A very simple SSh server that "boxes" you into a sandbox environment using
[uLinux](https://github.com/prologic/ulinux) `box` utility for creating
lightweight sandboxes.

> Based on [this example](https://github.com/gliderlabs/ssh/tree/master/_examples/ssh-pty)

## Usage

```#!console
sshbox ~/.ssh/authorized_keys 'box run alpine /bin/sh'
```

__NOTE:__ This only works on [uLinux](https://github.com/prologic/ulinux)
