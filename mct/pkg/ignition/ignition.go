package ignition

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	buconfig "github.com/coreos/butane/config"
	bucommon "github.com/coreos/butane/config/common"
)

const ignition = `
variant: fcos
version: 1.4.0
passwd:
  users:
  # core/core
    - name: core
      ssh_authorized_keys:
        - "{{.PubKey}}"
      password_hash: "$y$j9T$5J9zYWR/2iOJu//O5VVca.$ZKbQhIcgXchJYgBO0ZnGk4o0cCxn7GVZ1.CbYIh6uR0"

systemd:
  units:
    - name: serial-getty@ttyS0.service
      dropins:
      - name: autologin-mct.conf
        contents: |
          [Service]
          # Override Execstart in main unit
          ExecStart=
          # Add new Execstart with prefix to ignore failure
          ExecStart=-/usr/sbin/agetty --autologin mct --noclear %I $TERM
          TTYVTDisallocate=no

    - name: vsudd.service
      enabled: true
      contents: |
        [Unit]
        Description=VSOCK to unix domain socket forwarding. Forwards guest /var/run/docker.sock
        After=network-online.target
        Wants=network-online.target

        [Service]
        ExecStartPre=-/bin/podman kill vsudd
        ExecStartPre=-/bin/podman rm vsudd
        ExecStartPre=-/bin/podman pull quay.io/densityops/vsudd:f8cee7d
        ExecStart=/bin/podman run --name vsudd --privileged \
                    --volume /var/run:/var/run  \
                    quay.io/densityops/vsudd:f8cee7d -inport 2376:unix:/var/run/docker.sock
        ExecStop=/bin/podman stop vsudd

        [Install]
        WantedBy=multi-user.target
storage:
  files:
    - path: /etc/ssh/sshd_config.d/20-enable-passwords.conf
      overwrite: true
      mode: 0644
      contents:
        inline: |
          # Fedora CoreOS disables SSH password login by default.
          # Enable it.
          # This file must sort before 40-disable-passwords.conf.
          PasswordAuthentication yes
    - path: /etc/profile.d/systemd-pager.sh
      overwrite: true
      mode: 0644
      contents:
        inline: |
          # Tell systemd to not use a pager when printing information
          export SYSTEMD_PAGER=cat
    - path: /etc/sysctl.d/20-silence-audit.conf
      overwrite: true
      mode: 0644
      contents:
        inline: |
          # Raise console message logging level from DEBUG (7) to WARNING (4)
          # to hide audit messages from the interactive console
          kernel.printk=4
    - path: /etc/zincati/config.d/90-disable-auto-updates.toml
      overwrite: true
      mode: 0644
      contents:
        inline: |
          [updates]
          enabled = false
    - path: /etc/hostname
      overwrite: true
      mode: 0644
      contents:
        inline: |
          mct
`

type dynamicIgnition struct {
	PubKey string
}

func WriteIgnition(path, pubKey string) error {
	// add ssh-key
	d := &dynamicIgnition{
		PubKey: pubKey,
	}
	ignTemplate, err := template.New("ignition").Parse(ignition)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	if err := ignTemplate.Execute(&b, d); err != nil {
		return err
	}
	// create ign
	options := bucommon.TranslateBytesOptions{}
	options.FilesDir = "/tmp"
	ign, _, err := buconfig.TranslateBytes(b.Bytes(), options)
	if err != nil {
		return err
	}
	ignFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", path, err)
	}
	defer ignFile.Close()
	if _, err := ignFile.Write(append(ign, '\n')); err != nil {
		return fmt.Errorf("failed to write config to %s: %v", ignFile.Name(), err)
	}
	return nil
}
